//go:build integration

package groups_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/avatars"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

// Audit A-02: POST /groups/{id}/avatar wrote the image BEFORE checking who was
// asking. The caller got a 403, but the bytes under the group's (deterministic)
// avatar key were already replaced — any account could deface any group. The
// test drives the real HTTP handler, because the bug is the handler's order of
// operations, not either service on its own.

func squarePNG(t *testing.T, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestGroupAvatarIsNotStoredForSomeoneWhoMayNotChangeIt(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	gsvc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	avsvc := avatars.NewService(pool, avatars.NewPostgresRepository())

	levels := map[uuid.UUID]int{}
	seed := func(name string, level int) uuid.UUID {
		id := testdb.SeedUser(t, pool, name, level)
		levels[id] = level
		return id
	}
	founder := seed("founder", 5)
	outsider := seed("outsider", 0) // basic, not a member
	junior := seed("junior", 8)     // cannot even see a level-5 group

	// The actor comes from a test header instead of a JWT; everything after
	// that is the production handler.
	h := groups.NewHandler(gsvc, avsvc, func(r *http.Request) (groups.ActorMeta, bool) {
		id, err := uuid.Parse(r.Header.Get("X-Test-User"))
		if err != nil {
			return groups.ActorMeta{}, false
		}
		return groups.ActorMeta{UserID: id, RoleLevel: levels[id]}, true
	}, nil)
	router := chi.NewRouter()
	router.Route("/groups", h.Routes)

	upload := func(user, groupID uuid.UUID, img []byte) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/groups/"+groupID.String()+"/avatar", bytes.NewReader(img))
		req.Header.Set("X-Test-User", user.String())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr.Code
	}
	stored := func(groupID uuid.UUID) []byte {
		t.Helper()
		img, err := avsvc.Load(ctx, "group", groupID)
		if errors.Is(err, avatars.ErrNotFound) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return img.Bytes
	}

	community, err := gsvc.Create(ctx, groups.CreateInput{Name: "Open", Kind: groups.KindCommunity, IsPublic: true},
		groups.ActorMeta{UserID: founder, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	original := squarePNG(t, color.RGBA{0, 128, 0, 255})
	if code := upload(founder, community.ID, original); code != http.StatusOK {
		t.Fatalf("founder upload: want 200, got %d", code)
	}

	// A visible community, a caller who is not its founder: refused AND untouched.
	defaced := squarePNG(t, color.RGBA{255, 0, 0, 255})
	if code := upload(outsider, community.ID, defaced); code != http.StatusForbidden {
		t.Fatalf("outsider upload: want 403, got %d", code)
	}
	if got := stored(community.ID); !bytes.Equal(got, original) {
		t.Fatalf("a refused upload replaced the stored avatar (%d bytes, want the founder's %d)", len(got), len(original))
	}

	// A group the caller cannot see: 404, and nothing is written for it.
	hidden, err := gsvc.Create(ctx, groups.CreateInput{Name: "Level five", MinRoleLevel: 5},
		groups.ActorMeta{UserID: founder, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	if code := upload(junior, hidden.ID, defaced); code != http.StatusNotFound {
		t.Fatalf("upload to an invisible group: want 404, got %d", code)
	}
	if got := stored(hidden.ID); got != nil {
		t.Fatalf("a refused upload stored an avatar for an invisible group (%d bytes)", len(got))
	}

	// A group that does not exist at all: 404, and no orphan row.
	ghost := uuid.New()
	if code := upload(outsider, ghost, defaced); code != http.StatusNotFound {
		t.Fatalf("upload to a missing group: want 404, got %d", code)
	}
	if got := stored(ghost); got != nil {
		t.Fatalf("a refused upload stored an avatar for a group that does not exist (%d bytes)", len(got))
	}
}
