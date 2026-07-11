package chatfolders

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateName(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"Work", "Work", false},
		{"  Работа  ", "Работа", false},
		{"", "", true},
		{"   ", "", true},
		{strings.Repeat("ы", 64), strings.Repeat("ы", 64), false},
		{strings.Repeat("ы", 65), "", true},
	}
	for _, c := range cases {
		got, err := ValidateName(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ValidateName(%q): want error, got %q", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("ValidateName(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

// Validation short-circuits before any repository or authorizer call, so a
// zero-value service is enough for these paths.
func TestServiceValidation(t *testing.T) {
	svc := NewService(nil, nil, nil)
	ctx := context.Background()
	actor := Actor{UserID: uuid.New(), RoleLevel: 5}

	if _, err := svc.CreateFolder(ctx, actor, "  "); err != ErrValidation {
		t.Errorf("CreateFolder blank name: want ErrValidation, got %v", err)
	}
	if err := svc.RenameFolder(ctx, actor, uuid.New(), strings.Repeat("x", 65)); err != ErrValidation {
		t.Errorf("RenameFolder long name: want ErrValidation, got %v", err)
	}
	if err := svc.AddItem(ctx, actor, uuid.New(), Item{ChatType: "channel", ChatID: uuid.New()}); err != ErrValidation {
		t.Errorf("AddItem bad chat type: want ErrValidation, got %v", err)
	}
	if err := svc.Archive(ctx, actor, "channel", uuid.New()); err != ErrValidation {
		t.Errorf("Archive bad chat type: want ErrValidation, got %v", err)
	}
	if err := svc.ReorderFolders(ctx, actor, nil); err != ErrValidation {
		t.Errorf("ReorderFolders empty: want ErrValidation, got %v", err)
	}
	dup := uuid.New()
	if err := svc.ReorderFolders(ctx, actor, []uuid.UUID{dup, dup}); err != ErrValidation {
		t.Errorf("ReorderFolders duplicates: want ErrValidation, got %v", err)
	}
}
