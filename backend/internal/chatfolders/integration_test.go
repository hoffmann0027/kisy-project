//go:build integration

package chatfolders_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/chatfolders"
	"kisy-backend/internal/platform/testdb"
)

// allowAll authorizes every chat; denySet fails for the listed chats,
// simulating the access filter (masked as ErrNotFound by the service).
func allowAll(context.Context, string, uuid.UUID, uuid.UUID, int) error { return nil }

func denySet(denied map[uuid.UUID]bool) chatfolders.ChatAuthorizer {
	return func(_ context.Context, _ string, chatID, _ uuid.UUID, _ int) error {
		if denied[chatID] {
			return errors.New("no access")
		}
		return nil
	}
}

func TestFolderLifecycle(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	svc := chatfolders.NewService(pool, chatfolders.NewPostgresRepository(), allowAll)

	alice := chatfolders.Actor{UserID: testdb.SeedUser(t, pool, "alice", 3), RoleLevel: 3}

	work, err := svc.CreateFolder(ctx, alice, "Работа")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	personal, err := svc.CreateFolder(ctx, alice, "Личное")
	if err != nil {
		t.Fatalf("create 2: %v", err)
	}
	if personal.Position <= work.Position {
		t.Fatalf("positions should grow: %d then %d", work.Position, personal.Position)
	}

	// Rename.
	if err := svc.RenameFolder(ctx, alice, work.ID, "Проекты"); err != nil {
		t.Fatalf("rename: %v", err)
	}

	// Items: add twice (idempotent), then remove.
	chat := uuid.New()
	item := chatfolders.Item{ChatType: "private", ChatID: chat}
	if err := svc.AddItem(ctx, alice, work.ID, item); err != nil {
		t.Fatalf("add item: %v", err)
	}
	if err := svc.AddItem(ctx, alice, work.ID, item); err != nil {
		t.Fatalf("add item again: %v", err)
	}
	folders, err := svc.ListFolders(ctx, alice)
	if err != nil || len(folders) != 2 {
		t.Fatalf("list: %v, %v", folders, err)
	}
	if folders[0].Name != "Проекты" || len(folders[0].Items) != 1 || folders[0].Items[0].ChatID != chat {
		t.Fatalf("folder state: %+v", folders[0])
	}

	// Reorder: personal first.
	if err := svc.ReorderFolders(ctx, alice, []uuid.UUID{personal.ID, work.ID}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	folders, _ = svc.ListFolders(ctx, alice)
	if folders[0].ID != personal.ID {
		t.Fatalf("after reorder first = %v, want %v", folders[0].ID, personal.ID)
	}

	// Stale/partial reorder list is rejected.
	if err := svc.ReorderFolders(ctx, alice, []uuid.UUID{work.ID}); err != chatfolders.ErrValidation {
		t.Fatalf("partial reorder: want ErrValidation, got %v", err)
	}

	if err := svc.RemoveItem(ctx, alice, work.ID, item); err != nil {
		t.Fatalf("remove item: %v", err)
	}

	// Delete cascades items.
	if err := svc.DeleteFolder(ctx, alice, work.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	folders, _ = svc.ListFolders(ctx, alice)
	if len(folders) != 1 {
		t.Fatalf("after delete: %+v", folders)
	}
}

func TestFolderUserIsolation(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	svc := chatfolders.NewService(pool, chatfolders.NewPostgresRepository(), allowAll)

	alice := chatfolders.Actor{UserID: testdb.SeedUser(t, pool, "alice", 3), RoleLevel: 3}
	bob := chatfolders.Actor{UserID: testdb.SeedUser(t, pool, "bob", 8), RoleLevel: 8}

	folder, err := svc.CreateFolder(ctx, alice, "Секретная")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Bob cannot see, rename, delete, or fill Alice's folder — all masked 404.
	if list, _ := svc.ListFolders(ctx, bob); len(list) != 0 {
		t.Fatalf("bob sees alice's folders: %+v", list)
	}
	if err := svc.RenameFolder(ctx, bob, folder.ID, "Моя"); err != chatfolders.ErrNotFound {
		t.Fatalf("rename foreign: want ErrNotFound, got %v", err)
	}
	if err := svc.DeleteFolder(ctx, bob, folder.ID); err != chatfolders.ErrNotFound {
		t.Fatalf("delete foreign: want ErrNotFound, got %v", err)
	}
	if err := svc.AddItem(ctx, bob, folder.ID, chatfolders.Item{ChatType: "group", ChatID: uuid.New()}); err != chatfolders.ErrNotFound {
		t.Fatalf("add to foreign: want ErrNotFound, got %v", err)
	}
	// Reordering with someone else's folder id is rejected.
	if err := svc.ReorderFolders(ctx, bob, []uuid.UUID{folder.ID}); err != chatfolders.ErrValidation {
		t.Fatalf("reorder foreign: want ErrValidation, got %v", err)
	}
}

func TestInaccessibleChatMasked(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	hidden := uuid.New()
	svc := chatfolders.NewService(pool, chatfolders.NewPostgresRepository(), denySet(map[uuid.UUID]bool{hidden: true}))
	alice := chatfolders.Actor{UserID: testdb.SeedUser(t, pool, "alice", 5), RoleLevel: 5}

	folder, err := svc.CreateFolder(ctx, alice, "Папка")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Adding an inaccessible chat is a masked 404, never a 403.
	if err := svc.AddItem(ctx, alice, folder.ID, chatfolders.Item{ChatType: "group", ChatID: hidden}); err != chatfolders.ErrNotFound {
		t.Fatalf("add hidden: want ErrNotFound, got %v", err)
	}
	// Same for archiving.
	if err := svc.Archive(ctx, alice, "group", hidden); err != chatfolders.ErrNotFound {
		t.Fatalf("archive hidden: want ErrNotFound, got %v", err)
	}
}

func TestArchiveLifecycle(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	svc := chatfolders.NewService(pool, chatfolders.NewPostgresRepository(), allowAll)

	alice := chatfolders.Actor{UserID: testdb.SeedUser(t, pool, "alice", 3), RoleLevel: 3}
	bob := chatfolders.Actor{UserID: testdb.SeedUser(t, pool, "bob", 8), RoleLevel: 8}
	chat := uuid.New()

	// Archive twice (idempotent).
	if err := svc.Archive(ctx, alice, "private", chat); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if err := svc.Archive(ctx, alice, "private", chat); err != nil {
		t.Fatalf("archive again: %v", err)
	}

	list, err := svc.ListArchived(ctx, alice)
	if err != nil || len(list) != 1 || list[0].ChatID != chat {
		t.Fatalf("archived list: %+v, %v", list, err)
	}

	// Archive is personal: Bob's list is empty.
	if list, _ := svc.ListArchived(ctx, bob); len(list) != 0 {
		t.Fatalf("bob sees alice's archive: %+v", list)
	}

	if err := svc.Unarchive(ctx, alice, "private", chat); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	if list, _ := svc.ListArchived(ctx, alice); len(list) != 0 {
		t.Fatalf("after unarchive: %+v", list)
	}
}
