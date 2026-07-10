package e2ee

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// Repository is the persistence port for the E2EE directory and mailbox.
type Repository interface {
	UpsertDevice(ctx context.Context, q db.DBTX, d *Device) error
	GetDevice(ctx context.Context, q db.DBTX, id uuid.UUID) (*Device, error)
	ListDevices(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]Device, error)
	RevokeDevice(ctx context.Context, q db.DBTX, id, ownerID uuid.UUID, at time.Time) error

	AddKeyPackages(ctx context.Context, q db.DBTX, deviceID uuid.UUID, packages [][]byte) error
	// ClaimKeyPackages atomically consumes one unclaimed key package per
	// active device of userID, skipping excludeDevice (uuid.Nil = none).
	// Devices whose pool ran dry are skipped.
	ClaimKeyPackages(ctx context.Context, q db.DBTX, userID, excludeDevice uuid.UUID) ([]ClaimedKeyPackage, error)
	CountKeyPackages(ctx context.Context, q db.DBTX, deviceID uuid.UUID) (int, error)

	InsertGroupMessage(ctx context.Context, q db.DBTX, m *GroupMessage) error
	// ListChatHandshake returns commits/proposals of a chat created after
	// afterID (zero UUID = from the beginning), oldest first.
	ListChatHandshake(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, afterID uuid.UUID, limit int) ([]GroupMessage, error)
	// ListWelcomes returns unfetched welcomes addressed to one device.
	ListWelcomes(ctx context.Context, q db.DBTX, deviceID uuid.UUID) ([]GroupMessage, error)
	MarkWelcomeFetched(ctx context.Context, q db.DBTX, id, deviceID uuid.UUID, at time.Time) error

	PutBackup(ctx context.Context, q db.DBTX, userID uuid.UUID, blob, kdfParams []byte) error
	GetBackup(ctx context.Context, q db.DBTX, userID uuid.UUID) (*Backup, error)
	DeleteBackup(ctx context.Context, q db.DBTX, userID uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const deviceColumns = `id, user_id, name, ed25519_pub, signed_by, signature, created_at, revoked_at`

func scanDevice(row pgx.Row) (*Device, error) {
	var d Device
	err := row.Scan(&d.ID, &d.UserID, &d.Name, &d.Ed25519Pub, &d.SignedBy, &d.Signature, &d.CreatedAt, &d.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("e2ee: scan device: %w", err)
	}
	return &d, nil
}

func (r *PostgresRepository) UpsertDevice(ctx context.Context, q db.DBTX, d *Device) error {
	err := q.QueryRow(ctx, `
		INSERT INTO e2ee_devices (id, user_id, name, ed25519_pub, signed_by, signature)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
		RETURNING created_at`,
		d.ID, d.UserID, d.Name, d.Ed25519Pub, d.SignedBy, d.Signature,
	).Scan(&d.CreatedAt)
	if err != nil {
		return fmt.Errorf("e2ee: upsert device: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetDevice(ctx context.Context, q db.DBTX, id uuid.UUID) (*Device, error) {
	return scanDevice(q.QueryRow(ctx, `SELECT `+deviceColumns+` FROM e2ee_devices WHERE id = $1`, id))
}

func (r *PostgresRepository) ListDevices(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]Device, error) {
	rows, err := q.Query(ctx, `
		SELECT `+deviceColumns+` FROM e2ee_devices
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("e2ee: list devices: %w", err)
	}
	defer rows.Close()

	var out []Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) RevokeDevice(ctx context.Context, q db.DBTX, id, ownerID uuid.UUID, at time.Time) error {
	tag, err := q.Exec(ctx, `
		UPDATE e2ee_devices SET revoked_at = $3
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, ownerID, at)
	if err != nil {
		return fmt.Errorf("e2ee: revoke device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) AddKeyPackages(ctx context.Context, q db.DBTX, deviceID uuid.UUID, packages [][]byte) error {
	for _, kp := range packages {
		if _, err := q.Exec(ctx,
			`INSERT INTO e2ee_key_packages (device_id, key_package) VALUES ($1, $2)`,
			deviceID, kp,
		); err != nil {
			return fmt.Errorf("e2ee: add key package: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) ClaimKeyPackages(ctx context.Context, q db.DBTX, userID, excludeDevice uuid.UUID) ([]ClaimedKeyPackage, error) {
	// One package per active device, consumed atomically. FOR UPDATE SKIP
	// LOCKED keeps concurrent claimers from racing to the same row.
	rows, err := q.Query(ctx, `
		WITH picks AS (
			SELECT pick.id
			FROM e2ee_devices d
			CROSS JOIN LATERAL (
				SELECT id FROM e2ee_key_packages
				WHERE device_id = d.id AND consumed_at IS NULL
				ORDER BY created_at
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			) pick
			WHERE d.user_id = $1 AND d.revoked_at IS NULL AND d.id <> $2
		)
		UPDATE e2ee_key_packages kp SET consumed_at = now()
		FROM picks WHERE kp.id = picks.id
		RETURNING kp.device_id, kp.key_package`, userID, excludeDevice)
	if err != nil {
		return nil, fmt.Errorf("e2ee: claim key packages: %w", err)
	}
	defer rows.Close()

	var out []ClaimedKeyPackage
	for rows.Next() {
		var c ClaimedKeyPackage
		if err := rows.Scan(&c.DeviceID, &c.KeyPackage); err != nil {
			return nil, fmt.Errorf("e2ee: scan claimed package: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CountKeyPackages(ctx context.Context, q db.DBTX, deviceID uuid.UUID) (int, error) {
	var n int
	err := q.QueryRow(ctx,
		`SELECT count(*) FROM e2ee_key_packages WHERE device_id = $1 AND consumed_at IS NULL`,
		deviceID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("e2ee: count key packages: %w", err)
	}
	return n, nil
}

const groupMessageColumns = `id, chat_type, chat_id, kind, sender_device, recipient_device, payload, epoch, created_at`

func scanGroupMessage(row pgx.Row) (*GroupMessage, error) {
	var m GroupMessage
	err := row.Scan(&m.ID, &m.ChatType, &m.ChatID, &m.Kind, &m.SenderDevice, &m.RecipientDevice,
		&m.Payload, &m.Epoch, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("e2ee: scan group message: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) InsertGroupMessage(ctx context.Context, q db.DBTX, m *GroupMessage) error {
	err := q.QueryRow(ctx, `
		INSERT INTO e2ee_group_messages (chat_type, chat_id, kind, sender_device, recipient_device, payload, epoch)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		m.ChatType, m.ChatID, m.Kind, m.SenderDevice, m.RecipientDevice, m.Payload, m.Epoch,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("e2ee: insert group message: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListChatHandshake(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, afterID uuid.UUID, limit int) ([]GroupMessage, error) {
	rows, err := q.Query(ctx, `
		SELECT `+groupMessageColumns+` FROM e2ee_group_messages
		WHERE chat_type = $1 AND chat_id = $2 AND recipient_device IS NULL
		  AND ($3::uuid = '00000000-0000-0000-0000-000000000000'
		       OR created_at > (SELECT created_at FROM e2ee_group_messages WHERE id = $3))
		ORDER BY created_at, id
		LIMIT $4`, chatType, chatID, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("e2ee: list handshake: %w", err)
	}
	defer rows.Close()

	var out []GroupMessage
	for rows.Next() {
		m, err := scanGroupMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListWelcomes(ctx context.Context, q db.DBTX, deviceID uuid.UUID) ([]GroupMessage, error) {
	rows, err := q.Query(ctx, `
		SELECT `+groupMessageColumns+` FROM e2ee_group_messages
		WHERE recipient_device = $1 AND fetched_at IS NULL
		ORDER BY created_at`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("e2ee: list welcomes: %w", err)
	}
	defer rows.Close()

	var out []GroupMessage
	for rows.Next() {
		m, err := scanGroupMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) MarkWelcomeFetched(ctx context.Context, q db.DBTX, id, deviceID uuid.UUID, at time.Time) error {
	tag, err := q.Exec(ctx, `
		UPDATE e2ee_group_messages SET fetched_at = $3
		WHERE id = $1 AND recipient_device = $2 AND fetched_at IS NULL`, id, deviceID, at)
	if err != nil {
		return fmt.Errorf("e2ee: mark welcome fetched: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) PutBackup(ctx context.Context, q db.DBTX, userID uuid.UUID, blob, kdfParams []byte) error {
	_, err := q.Exec(ctx, `
		INSERT INTO e2ee_backups (user_id, blob, kdf_params, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE
		SET blob = EXCLUDED.blob, kdf_params = EXCLUDED.kdf_params, updated_at = now()`,
		userID, blob, kdfParams)
	if err != nil {
		return fmt.Errorf("e2ee: put backup: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetBackup(ctx context.Context, q db.DBTX, userID uuid.UUID) (*Backup, error) {
	var b Backup
	err := q.QueryRow(ctx,
		`SELECT blob, kdf_params, updated_at FROM e2ee_backups WHERE user_id = $1`, userID,
	).Scan(&b.Blob, &b.KDFParams, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("e2ee: get backup: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) DeleteBackup(ctx context.Context, q db.DBTX, userID uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM e2ee_backups WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("e2ee: delete backup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
