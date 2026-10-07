package users

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/avatars"
)

type failingStore struct{ err error }

func (f failingStore) Store(context.Context, string, uuid.UUID, []byte) (string, error) {
	return "", f.err
}

// Audit A-42: a failed avatar upload answered with err.Error(), so a database
// or object-storage failure — pgx text, the S3 endpoint and bucket — went
// straight to the client. Only a problem with the image itself is the
// uploader's to read.
func TestAvatarUploadNamesNoInternals(t *testing.T) {
	upload := func(storeErr error) *httptest.ResponseRecorder {
		h := NewHandler(nil, failingStore{err: storeErr},
			func(*http.Request) (Identity, bool) { return Identity{UserID: uuid.New()}, true },
			func(*http.Request) ActorMeta { return ActorMeta{} })
		req := httptest.NewRequest(http.MethodPost, "/users/me/avatar", bytes.NewReader([]byte{1, 2, 3}))
		rec := httptest.NewRecorder()
		h.uploadAvatar(rec, req)
		return rec
	}

	internal := errors.New(`blobstore: put "avatars/x": dial tcp 10.0.4.17:9000: connect: connection refused (bucket kisy-private)`)
	rec := upload(internal)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("a storage failure answered %d, want 500", rec.Code)
	}
	for _, leak := range []string{"10.0.4.17", "kisy-private", "blobstore", "dial tcp"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Fatalf("the response names an internal detail %q: %s", leak, rec.Body.String())
		}
	}

	rec = upload(avatars.ErrNotSquare)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "square") {
		t.Fatalf("a problem with the image must be explained: %d %s", rec.Code, rec.Body.String())
	}
}
