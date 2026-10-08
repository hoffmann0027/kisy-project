//go:build integration

package announcements_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/announcements"
)

// "New Update": the CEO tells everyone a new version is out, what changed and
// where to get it.
func TestTheCEOAnnouncesANewVersionToEveryone(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, e.basic2); err != nil {
		t.Fatal(err)
	}

	r, err := e.svc.SendRelease(ctx, actor(e.ceo), announcements.ReleaseInput{
		Version: " 1.4.0 ", Notes: "Зум картинок, новые отзывы", DownloadURL: "https://kisy.onrender.com/app.apk",
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != "1.4.0" || r.RecipientCount != 5 { // everyone active but the CEO
		t.Fatalf("release: %+v", r)
	}
	var n int
	if err := e.pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications
		WHERE type = 'app_release' AND payload->>'version' = '1.4.0' AND payload->>'downloadUrl' IS NOT NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("%d notifications", n)
	}
	for _, id := range []uuid.UUID{e.exec, e.manager, e.basic1} {
		if e.live.created[id] != 1 {
			t.Fatalf("%s got no live event", id)
		}
	}

	list, err := e.svc.Releases(ctx, 0)
	if err != nil || len(list) != 1 || list[0].ID != r.ID {
		t.Fatalf("releases: %+v %v", list, err)
	}
}

func TestOnlyTheCEOAnnouncesVersions(t *testing.T) {
	e := setup(t)
	in := announcements.ReleaseInput{Version: "1.4.0", Notes: "Изменения"}
	for _, who := range []uuid.UUID{e.exec, e.director, e.basic1} {
		if _, err := e.svc.SendRelease(context.Background(), actor(who), in); !errors.Is(err, announcements.ErrNotCEO) {
			t.Fatalf("got %v, want ErrNotCEO", err)
		}
	}
}

func TestAReleaseNeedsAVersionNotesAndASafeLink(t *testing.T) {
	e := setup(t)
	for name, in := range map[string]announcements.ReleaseInput{
		"no version":   {Notes: "n"},
		"no notes":     {Version: "1.0"},
		"plain http":   {Version: "1.0", Notes: "n", DownloadURL: "http://example.com/a.apk"},
		"javascript":   {Version: "1.0", Notes: "n", DownloadURL: "javascript:alert(1)"},
		"no host":      {Version: "1.0", Notes: "n", DownloadURL: "https://"},
		"long version": {Version: "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0", Notes: "n"},
	} {
		if _, err := e.svc.SendRelease(context.Background(), actor(e.ceo), in); !errors.Is(err, announcements.ErrValidation) {
			t.Fatalf("%s: got %v, want ErrValidation", name, err)
		}
	}
}
