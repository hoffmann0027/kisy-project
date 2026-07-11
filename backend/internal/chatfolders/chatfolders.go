// Package chatfolders implements per-user chat folders and the chat archive
// (UPD3 stage H). Both are personal organizational metadata over chat
// references: they never grant, reveal, or widen access. The chat list is
// always produced by the access-filtered chats/groups services, so a folder
// item pointing at a chat the user can no longer see simply never surfaces.
package chatfolders

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

var (
	ErrValidation = errors.New("chatfolders: validation failed")
	// ErrNotFound covers both "does not exist" and "not yours" — the two are
	// indistinguishable to the caller by design (masked 404).
	ErrNotFound = errors.New("chatfolders: not found")
)

const (
	maxNameLen = 64
	maxFolders = 50
)

// ChatAuthorizer confirms the actor may access a chat. Injected to avoid
// import cycles (same shape as favorites.ChatAuthorizer).
type ChatAuthorizer func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, actorLevel int) error

// Actor identifies the acting user.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

// Item is one chat reference inside a folder.
type Item struct {
	ChatType string    `json:"chatType"`
	ChatID   uuid.UUID `json:"chatId"`
}

// Folder is a user's folder with its items.
type Folder struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Position int       `json:"position"`
	Items    []Item    `json:"items"`
}

// ArchiveEntry is one archived chat reference.
type ArchiveEntry struct {
	ChatType   string    `json:"chatType"`
	ChatID     uuid.UUID `json:"chatId"`
	ArchivedAt string    `json:"archivedAt"`
}

// Repository is the persistence port for folders and the archive. Every
// method scopes by user_id: one user can never touch another's rows.
type Repository interface {
	CreateFolder(ctx context.Context, q db.DBTX, userID uuid.UUID, name string) (Folder, error)
	RenameFolder(ctx context.Context, q db.DBTX, userID, folderID uuid.UUID, name string) (bool, error)
	DeleteFolder(ctx context.Context, q db.DBTX, userID, folderID uuid.UUID) (bool, error)
	CountFolders(ctx context.Context, q db.DBTX, userID uuid.UUID) (int, error)
	// ReorderFolders assigns positions 0..n-1 following orderedIDs. Returns
	// false if any id is missing or not owned by the user.
	ReorderFolders(ctx context.Context, q db.DBTX, userID uuid.UUID, orderedIDs []uuid.UUID) (bool, error)
	ListFolders(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]Folder, error)
	FolderOwned(ctx context.Context, q db.DBTX, userID, folderID uuid.UUID) (bool, error)
	AddItem(ctx context.Context, q db.DBTX, folderID uuid.UUID, item Item) error
	RemoveItem(ctx context.Context, q db.DBTX, folderID uuid.UUID, item Item) error

	Archive(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error
	Unarchive(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error
	ListArchived(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]ArchiveEntry, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) CreateFolder(ctx context.Context, q db.DBTX, userID uuid.UUID, name string) (Folder, error) {
	var f Folder
	err := q.QueryRow(ctx, `
		INSERT INTO chat_folders (user_id, name, position)
		VALUES ($1, $2, COALESCE((SELECT MAX(position) + 1 FROM chat_folders WHERE user_id = $1), 0))
		RETURNING id, name, position`,
		userID, name).Scan(&f.ID, &f.Name, &f.Position)
	if err != nil {
		return Folder{}, fmt.Errorf("chatfolders: create: %w", err)
	}
	f.Items = []Item{}
	return f, nil
}

func (r *PostgresRepository) RenameFolder(ctx context.Context, q db.DBTX, userID, folderID uuid.UUID, name string) (bool, error) {
	tag, err := q.Exec(ctx, `UPDATE chat_folders SET name = $3 WHERE id = $2 AND user_id = $1`,
		userID, folderID, name)
	if err != nil {
		return false, fmt.Errorf("chatfolders: rename: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) DeleteFolder(ctx context.Context, q db.DBTX, userID, folderID uuid.UUID) (bool, error) {
	tag, err := q.Exec(ctx, `DELETE FROM chat_folders WHERE id = $2 AND user_id = $1`, userID, folderID)
	if err != nil {
		return false, fmt.Errorf("chatfolders: delete: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) CountFolders(ctx context.Context, q db.DBTX, userID uuid.UUID) (int, error) {
	var n int
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM chat_folders WHERE user_id = $1`, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("chatfolders: count: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) ReorderFolders(ctx context.Context, q db.DBTX, userID uuid.UUID, orderedIDs []uuid.UUID) (bool, error) {
	// Single statement: position = index in the provided array. Rows affected
	// must equal both the array length and the user's folder count, otherwise
	// the list was stale or referenced someone else's folder.
	tag, err := q.Exec(ctx, `
		UPDATE chat_folders f
		SET position = u.ord - 1
		FROM UNNEST($2::uuid[]) WITH ORDINALITY AS u(id, ord)
		WHERE f.id = u.id AND f.user_id = $1`,
		userID, orderedIDs)
	if err != nil {
		return false, fmt.Errorf("chatfolders: reorder: %w", err)
	}
	if int(tag.RowsAffected()) != len(orderedIDs) {
		return false, nil
	}
	total, err := r.CountFolders(ctx, q, userID)
	if err != nil {
		return false, err
	}
	return total == len(orderedIDs), nil
}

func (r *PostgresRepository) ListFolders(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]Folder, error) {
	rows, err := q.Query(ctx, `
		SELECT id, name, position FROM chat_folders
		WHERE user_id = $1 ORDER BY position, created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("chatfolders: list: %w", err)
	}
	defer rows.Close()

	var out []Folder
	index := make(map[uuid.UUID]int)
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.Position); err != nil {
			return nil, fmt.Errorf("chatfolders: scan folder: %w", err)
		}
		f.Items = []Item{}
		index[f.ID] = len(out)
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	itemRows, err := q.Query(ctx, `
		SELECT i.folder_id, i.chat_type, i.chat_id
		FROM chat_folder_items i
		JOIN chat_folders f ON f.id = i.folder_id
		WHERE f.user_id = $1
		ORDER BY i.created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("chatfolders: list items: %w", err)
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var folderID uuid.UUID
		var it Item
		if err := itemRows.Scan(&folderID, &it.ChatType, &it.ChatID); err != nil {
			return nil, fmt.Errorf("chatfolders: scan item: %w", err)
		}
		if i, ok := index[folderID]; ok {
			out[i].Items = append(out[i].Items, it)
		}
	}
	return out, itemRows.Err()
}

func (r *PostgresRepository) FolderOwned(ctx context.Context, q db.DBTX, userID, folderID uuid.UUID) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM chat_folders WHERE id = $2 AND user_id = $1)`,
		userID, folderID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("chatfolders: owned: %w", err)
	}
	return ok, nil
}

func (r *PostgresRepository) AddItem(ctx context.Context, q db.DBTX, folderID uuid.UUID, item Item) error {
	_, err := q.Exec(ctx, `
		INSERT INTO chat_folder_items (folder_id, chat_type, chat_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`,
		folderID, item.ChatType, item.ChatID)
	if err != nil {
		return fmt.Errorf("chatfolders: add item: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RemoveItem(ctx context.Context, q db.DBTX, folderID uuid.UUID, item Item) error {
	_, err := q.Exec(ctx, `
		DELETE FROM chat_folder_items WHERE folder_id = $1 AND chat_type = $2 AND chat_id = $3`,
		folderID, item.ChatType, item.ChatID)
	if err != nil {
		return fmt.Errorf("chatfolders: remove item: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Archive(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO chat_archives (user_id, chat_type, chat_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, chat_type, chat_id) DO NOTHING`,
		userID, chatType, chatID)
	if err != nil {
		return fmt.Errorf("chatfolders: archive: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Unarchive(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		DELETE FROM chat_archives WHERE user_id = $1 AND chat_type = $2 AND chat_id = $3`,
		userID, chatType, chatID)
	if err != nil {
		return fmt.Errorf("chatfolders: unarchive: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListArchived(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]ArchiveEntry, error) {
	rows, err := q.Query(ctx, `
		SELECT chat_type, chat_id, archived_at::text FROM chat_archives
		WHERE user_id = $1 ORDER BY archived_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("chatfolders: list archived: %w", err)
	}
	defer rows.Close()
	var out []ArchiveEntry
	for rows.Next() {
		var e ArchiveEntry
		if err := rows.Scan(&e.ChatType, &e.ChatID, &e.ArchivedAt); err != nil {
			return nil, fmt.Errorf("chatfolders: scan archived: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Service holds folder/archive operations for the acting user.
type Service struct {
	pool      *pgxpool.Pool
	repo      Repository
	authorize ChatAuthorizer
}

func NewService(pool *pgxpool.Pool, repo Repository, authorize ChatAuthorizer) *Service {
	return &Service{pool: pool, repo: repo, authorize: authorize}
}

// ValidateName normalizes and validates a folder name.
func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > maxNameLen {
		return "", ErrValidation
	}
	return name, nil
}

func validChatType(t string) bool { return t == "private" || t == "group" }

func (s *Service) CreateFolder(ctx context.Context, actor Actor, name string) (Folder, error) {
	name, err := ValidateName(name)
	if err != nil {
		return Folder{}, err
	}
	n, err := s.repo.CountFolders(ctx, s.pool, actor.UserID)
	if err != nil {
		return Folder{}, err
	}
	if n >= maxFolders {
		return Folder{}, ErrValidation
	}
	return s.repo.CreateFolder(ctx, s.pool, actor.UserID, name)
}

func (s *Service) RenameFolder(ctx context.Context, actor Actor, folderID uuid.UUID, name string) error {
	name, err := ValidateName(name)
	if err != nil {
		return err
	}
	ok, err := s.repo.RenameFolder(ctx, s.pool, actor.UserID, folderID, name)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeleteFolder(ctx context.Context, actor Actor, folderID uuid.UUID) error {
	ok, err := s.repo.DeleteFolder(ctx, s.pool, actor.UserID, folderID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ReorderFolders(ctx context.Context, actor Actor, orderedIDs []uuid.UUID) error {
	if len(orderedIDs) == 0 || len(orderedIDs) > maxFolders {
		return ErrValidation
	}
	seen := make(map[uuid.UUID]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if _, dup := seen[id]; dup {
			return ErrValidation
		}
		seen[id] = struct{}{}
	}
	ok, err := s.repo.ReorderFolders(ctx, s.pool, actor.UserID, orderedIDs)
	if err != nil {
		return err
	}
	if !ok {
		return ErrValidation
	}
	return nil
}

func (s *Service) ListFolders(ctx context.Context, actor Actor) ([]Folder, error) {
	list, err := s.repo.ListFolders(ctx, s.pool, actor.UserID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Folder{}
	}
	return list, nil
}

// AddItem places a chat into the actor's folder. Chat access is verified so
// the folder never accumulates references the user cannot see; an
// authorization failure surfaces as ErrNotFound (masked — existence of an
// inaccessible chat is never confirmed).
func (s *Service) AddItem(ctx context.Context, actor Actor, folderID uuid.UUID, item Item) error {
	if !validChatType(item.ChatType) {
		return ErrValidation
	}
	owned, err := s.repo.FolderOwned(ctx, s.pool, actor.UserID, folderID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	if err := s.authorize(ctx, item.ChatType, item.ChatID, actor.UserID, actor.RoleLevel); err != nil {
		return ErrNotFound
	}
	return s.repo.AddItem(ctx, s.pool, folderID, item)
}

// RemoveItem drops a chat reference. No chat authorization is needed to
// forget a chat (mirrors favorites.Remove).
func (s *Service) RemoveItem(ctx context.Context, actor Actor, folderID uuid.UUID, item Item) error {
	if !validChatType(item.ChatType) {
		return ErrValidation
	}
	owned, err := s.repo.FolderOwned(ctx, s.pool, actor.UserID, folderID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	return s.repo.RemoveItem(ctx, s.pool, folderID, item)
}

// Archive hides a chat into the actor's personal archive. Access is checked
// with the same 404 mask as AddItem.
func (s *Service) Archive(ctx context.Context, actor Actor, chatType string, chatID uuid.UUID) error {
	if !validChatType(chatType) {
		return ErrValidation
	}
	if err := s.authorize(ctx, chatType, chatID, actor.UserID, actor.RoleLevel); err != nil {
		return ErrNotFound
	}
	return s.repo.Archive(ctx, s.pool, actor.UserID, chatType, chatID)
}

func (s *Service) Unarchive(ctx context.Context, actor Actor, chatType string, chatID uuid.UUID) error {
	if !validChatType(chatType) {
		return ErrValidation
	}
	return s.repo.Unarchive(ctx, s.pool, actor.UserID, chatType, chatID)
}

func (s *Service) ListArchived(ctx context.Context, actor Actor) ([]ArchiveEntry, error) {
	list, err := s.repo.ListArchived(ctx, s.pool, actor.UserID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []ArchiveEntry{}
	}
	return list, nil
}
