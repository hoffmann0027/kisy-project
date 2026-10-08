//go:build integration

package users_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/users"
)

// The CEO's user table ("Пользователи" in the admin panel) loaded the newest
// hundred accounts and nothing else: anyone further down could not be found
// at all. It now searches and filters on the server, like "Верификация".
func TestAdminUserListFilters(t *testing.T) {
	_, repo, pool := newUsers(t)
	ctx := context.Background()
	director := testdb.SeedUser(t, pool, "director_anna", 3)
	manager := testdb.SeedUser(t, pool, "manager_boris", 5)
	basic := testdb.SeedUser(t, pool, "basic_vera", 0)
	off := testdb.SeedUser(t, pool, "manager_off", 5)
	if _, err := pool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, off); err != nil {
		t.Fatal(err)
	}

	list := func(f users.ListFilter) map[uuid.UUID]bool {
		t.Helper()
		got, err := repo.List(ctx, pool, f, 100, 0)
		if err != nil {
			t.Fatal(err)
		}
		out := map[uuid.UUID]bool{}
		for _, u := range got {
			out[u.ID] = true
		}
		return out
	}
	want := func(name string, got map[uuid.UUID]bool, ids ...uuid.UUID) {
		t.Helper()
		if len(got) != len(ids) {
			t.Fatalf("%s: %d accounts, want %d", name, len(got), len(ids))
		}
		for _, id := range ids {
			if !got[id] {
				t.Fatalf("%s: missing %s", name, id)
			}
		}
	}
	yes, no := true, false

	want("everyone", list(users.ListFilter{}), director, manager, basic, off)
	want("login prefix", list(users.ListFilter{Query: "MANAGER_"}), manager, off)
	want("name, any part", list(users.ListFilter{Query: testdb.SeedDisplayName("basic_vera")}), basic)
	want("level 5", list(users.ListFilter{Level: 5}), manager, off)
	want("basic", list(users.ListFilter{Basic: true}), basic)
	want("active", list(users.ListFilter{Active: &yes}), director, manager, basic)
	want("switched off", list(users.ListFilter{Active: &no}), off)
	want("level 5, active, by login", list(users.ListFilter{Query: "manager", Level: 5, Active: &yes}), manager)
	want("nobody", list(users.ListFilter{Query: "no_such_person"}))
}
