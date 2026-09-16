//go:build integration

package users_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/users"
)

func newUsers(t *testing.T) (*users.Service, *users.PostgresRepository, *pgxpool.Pool) {
	pool := testdb.New(t)
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	repo := users.NewPostgresRepository()
	return users.NewService(pool, repo, rec), repo, pool
}

func rename(t *testing.T, svc *users.Service, id uuid.UUID, name string) error {
	t.Helper()
	_, err := svc.ChangeDisplayName(context.Background(), id, name, users.ActorMeta{})
	return err
}

// Unique ignoring case and repeated spaces — enforced by the database, so it
// holds for any path that writes a name, not only for the one tested here.
func TestDisplayNameUniqueIgnoringCaseAndSpaces(t *testing.T) {
	svc, _, pool := newUsers(t)
	anna := testdb.SeedUser(t, pool, "anna", 5)
	other := testdb.SeedUser(t, pool, "other", 5)

	if err := rename(t, svc, anna, "Анна Смирнова"); err != nil {
		t.Fatalf("first holder: %v", err)
	}
	for _, clash := range []string{"анна смирнова", "АННА СМИРНОВА", "  Анна    Смирнова ", "аННа сМИРнова"} {
		if err := rename(t, svc, other, clash); !errors.Is(err, users.ErrDisplayNameTaken) {
			t.Errorf("rename to %q: got %v, want ErrDisplayNameTaken", clash, err)
		}
	}
	// Latin is folded too, and a genuinely different name is fine.
	if err := rename(t, svc, anna, "Anna Smith"); err != nil {
		t.Fatal(err)
	}
	if err := rename(t, svc, other, "ANNA SMITH"); !errors.Is(err, users.ErrDisplayNameTaken) {
		t.Errorf("latin case: got %v, want ErrDisplayNameTaken", err)
	}
	if err := rename(t, svc, other, "Анна Смирнова"); err != nil {
		t.Errorf("the name freed by the first holder: %v", err)
	}
}

func TestDisplayNameRuleOnRename(t *testing.T) {
	svc, repo, pool := newUsers(t)
	id := testdb.SeedUser(t, pool, "someone", 5)

	if err := rename(t, svc, id, "hamza_1"); !errors.Is(err, users.ErrDisplayNameCharacters) {
		t.Fatalf("digits and underscore: got %v", err)
	}
	if err := rename(t, svc, id, "Я"); !errors.Is(err, users.ErrDisplayNameLength) {
		t.Fatalf("one letter: got %v", err)
	}
	if err := rename(t, svc, id, "  Пётр   Первый "); err != nil {
		t.Fatal(err)
	}
	u, err := repo.GetByID(context.Background(), pool, id)
	if err != nil {
		t.Fatal(err)
	}
	if u.DisplayName != "Пётр Первый" {
		t.Fatalf("stored name not normalized: %q", u.DisplayName)
	}
}

// The database refuses a bad name even from a path that skipped the Go check.
func TestDatabaseRefusesAnInvalidName(t *testing.T) {
	_, _, pool := newUsers(t)
	id := testdb.SeedUser(t, pool, "direct", 5)
	_, err := pool.Exec(context.Background(), `UPDATE users SET display_name = 'bad_name_1' WHERE id = $1`, id)
	if err == nil {
		t.Fatal("an invalid unflagged name must violate users_display_name_valid")
	}
}

// A flagged account (migration 46) keeps its old name until it picks one,
// and picking one clears the flag.
func TestChoosingANameClearsTheFlag(t *testing.T) {
	svc, repo, pool := newUsers(t)
	ctx := context.Background()
	holder := testdb.SeedUser(t, pool, "holder", 5)
	late := testdb.SeedUser(t, pool, "late", 5)
	if err := rename(t, svc, holder, "Иван Петров"); err != nil {
		t.Fatal(err)
	}
	// What the migration does to a later account with the same name.
	if _, err := pool.Exec(ctx,
		`UPDATE users SET display_name_needs_change = true, display_name = 'иван   петров' WHERE id = $1`, late); err != nil {
		t.Fatalf("a flagged duplicate must be storable: %v", err)
	}
	u, _ := repo.GetByID(ctx, pool, late)
	if !u.DisplayNameNeedsChange || !u.ToDTO().DisplayNameNeedsChange {
		t.Fatal("the flag must reach the client")
	}

	if err := rename(t, svc, late, "Иван Петров"); !errors.Is(err, users.ErrDisplayNameTaken) {
		t.Fatalf("picking the same name again: got %v", err)
	}
	if u, _ = repo.GetByID(ctx, pool, late); !u.DisplayNameNeedsChange {
		t.Fatal("a refused rename must leave the flag in place")
	}
	if err := rename(t, svc, late, "Иван Сидоров"); err != nil {
		t.Fatal(err)
	}
	if u, _ = repo.GetByID(ctx, pool, late); u.DisplayNameNeedsChange {
		t.Fatal("a successful rename must clear the flag")
	}
}

// A basic account finds people by the whole name — and "whole" means the same
// normalized value uniqueness uses, so case and spacing do not hide anyone.
func TestFullNameSearchUsesTheNormalizedName(t *testing.T) {
	svc, repo, pool := newUsers(t)
	ctx := context.Background()
	seeker := testdb.SeedUser(t, pool, "seeker", 0)
	target := testdb.SeedUser(t, pool, "target", 5)
	if err := rename(t, svc, target, "Мария Иванова"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"Мария Иванова", "мария иванова", "  МАРИЯ   иванова "} {
		found, err := repo.Search(ctx, pool, seeker, 0, q, 20)
		if err != nil {
			t.Fatal(err)
		}
		if len(found) != 1 || found[0].ID != target {
			t.Errorf("search %q: got %d results", q, len(found))
		}
	}
	if found, _ := repo.Search(ctx, pool, seeker, 0, "Мария", 20); len(found) != 0 {
		t.Error("a partial name must not match")
	}
}

// lower() folds Cyrillic only under a Unicode-aware locale, and the locale of
// a production database is not under this code's control. Under the C
// collation — ASCII folding only — the key must still fold Cyrillic.
func TestNameKeyFoldsCyrillicUnderTheCLocale(t *testing.T) {
	_, _, pool := newUsers(t)
	var key string
	err := pool.QueryRow(context.Background(),
		`SELECT kisy_display_name_key('  ЁЛКИН   ЇЖАК Łukasz ' COLLATE "C")`).Scan(&key)
	if err != nil {
		t.Fatal(err)
	}
	if key != "ёлкин їжак Łukasz" && key != "ёлкин їжак łukasz" {
		t.Fatalf("key under C collation = %q", key)
	}
}
