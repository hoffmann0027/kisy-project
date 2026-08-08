// Command blobmigrate moves already-stored file bytes between Postgres and
// object storage, one batch at a time, while the application keeps serving:
// reads are dual-path, so a row is readable before, during and after its move.
//
//	blobmigrate                 # move DB bytes -> object storage
//	blobmigrate -direction=to-db  # move them back (e.g. before rolling back)
//	blobmigrate -dry-run          # report what would move
//
// It is idempotent and resumable: each pass picks rows that still need moving,
// so re-running after an interruption simply continues.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/config"
	"kisy-backend/internal/platform/blobstore"
	"kisy-backend/internal/platform/logger"
	"kisy-backend/internal/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("blobmigrate failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	direction := flag.String("direction", "to-store", "to-store | to-db")
	batch := flag.Int("batch", 50, "rows per batch")
	dryRun := flag.Bool("dry-run", false, "report what would move, change nothing")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel)

	if !cfg.Blob.Enabled() {
		return fmt.Errorf("object storage is not configured (set BLOB_S3_ENDPOINT/BLOB_S3_BUCKET)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN(), postgres.PoolSettings(cfg.DBPool))
	if err != nil {
		return err
	}
	defer pool.Close()

	store, err := blobstore.NewS3(ctx, blobstore.Config{
		Endpoint:  cfg.Blob.Endpoint,
		Bucket:    cfg.Blob.Bucket,
		AccessKey: cfg.Blob.AccessKey,
		SecretKey: cfg.Blob.SecretKey,
		Region:    cfg.Blob.Region,
		UseSSL:    cfg.Blob.UseSSL,
	})
	if err != nil {
		return err
	}

	m := migrator{pool: pool, store: store, log: log, batch: *batch, dryRun: *dryRun}
	switch *direction {
	case "to-store":
		return m.toStore(ctx)
	case "to-db":
		return m.toDB(ctx)
	default:
		return fmt.Errorf("unknown -direction %q (want to-store or to-db)", *direction)
	}
}

type migrator struct {
	pool   *pgxpool.Pool
	store  blobstore.Store
	log    *slog.Logger
	batch  int
	dryRun bool
}

func (m migrator) toStore(ctx context.Context) error {
	if err := m.attachmentsToStore(ctx); err != nil {
		return err
	}
	return m.avatarsToStore(ctx)
}

// attachmentsToStore walks attachments whose bytes are still inline. Each row
// is written to the store BEFORE the column is cleared, and the update is
// per-row, so an interruption can only leave a row whose bytes exist in both
// places — harmless, and the next pass finishes it.
func (m migrator) attachmentsToStore(ctx context.Context) error {
	moved, bytesMoved := 0, int64(0)
	for {
		rows, err := m.pool.Query(ctx, `
			SELECT id, mime_type, data FROM attachments
			WHERE data IS NOT NULL AND storage_path = ''
			LIMIT $1`, m.batch)
		if err != nil {
			return fmt.Errorf("blobmigrate: select attachments: %w", err)
		}

		type item struct {
			id   uuid.UUID
			mime string
			data []byte
		}
		var batch []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.mime, &it.data); err != nil {
				rows.Close()
				return fmt.Errorf("blobmigrate: scan attachment: %w", err)
			}
			batch = append(batch, it)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		for _, it := range batch {
			key := "attachments/" + it.id.String()[:2] + "/" + it.id.String()
			if m.dryRun {
				moved++
				bytesMoved += int64(len(it.data))
				continue
			}
			if err := m.store.Put(ctx, key, it.data, it.mime); err != nil {
				return err
			}
			if _, err := m.pool.Exec(ctx,
				`UPDATE attachments SET storage_path = $2, data = NULL WHERE id = $1`, it.id, key); err != nil {
				return fmt.Errorf("blobmigrate: update attachment %s: %w", it.id, err)
			}
			moved++
			bytesMoved += int64(len(it.data))
		}
		m.log.Info("attachments batch moved", "total", moved, "bytes", bytesMoved, "dryRun", m.dryRun)

		if m.dryRun {
			break // nothing changes, so the same batch would repeat forever
		}
	}
	m.log.Info("attachments done", "moved", moved, "bytes", bytesMoved, "dryRun", m.dryRun)
	return nil
}

func (m migrator) avatarsToStore(ctx context.Context) error {
	moved := 0
	for {
		rows, err := m.pool.Query(ctx, `
			SELECT owner_type, owner_id, content_type, bytes FROM avatars
			WHERE bytes IS NOT NULL AND storage_path = ''
			LIMIT $1`, m.batch)
		if err != nil {
			return fmt.Errorf("blobmigrate: select avatars: %w", err)
		}

		type item struct {
			ownerType string
			ownerID   uuid.UUID
			mime      string
			data      []byte
		}
		var batch []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.ownerType, &it.ownerID, &it.mime, &it.data); err != nil {
				rows.Close()
				return fmt.Errorf("blobmigrate: scan avatar: %w", err)
			}
			batch = append(batch, it)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		for _, it := range batch {
			key := "avatars/" + it.ownerType + "/" + it.ownerID.String()
			if m.dryRun {
				moved++
				continue
			}
			if err := m.store.Put(ctx, key, it.data, it.mime); err != nil {
				return err
			}
			// The schema's XOR check means path and bytes must flip together.
			if _, err := m.pool.Exec(ctx,
				`UPDATE avatars SET storage_path = $3, bytes = NULL
				 WHERE owner_type = $1 AND owner_id = $2`, it.ownerType, it.ownerID, key); err != nil {
				return fmt.Errorf("blobmigrate: update avatar %s/%s: %w", it.ownerType, it.ownerID, err)
			}
			moved++
		}
		if m.dryRun {
			break
		}
	}
	m.log.Info("avatars done", "moved", moved, "dryRun", m.dryRun)
	return nil
}

// toDB is the reverse path, needed before rolling migration 000040 back or
// when abandoning object storage.
func (m migrator) toDB(ctx context.Context) error {
	moved := 0
	for {
		rows, err := m.pool.Query(ctx, `
			SELECT id, storage_path FROM attachments
			WHERE data IS NULL AND storage_path <> '' LIMIT $1`, m.batch)
		if err != nil {
			return fmt.Errorf("blobmigrate: select attachments back: %w", err)
		}
		type item struct {
			id  uuid.UUID
			key string
		}
		var batch []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.key); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, it)
		}
		rows.Close()
		if len(batch) == 0 {
			break
		}
		for _, it := range batch {
			data, err := m.store.Get(ctx, it.key)
			if err != nil {
				return err
			}
			if m.dryRun {
				moved++
				continue
			}
			if _, err := m.pool.Exec(ctx,
				`UPDATE attachments SET data = $2, storage_path = '' WHERE id = $1`, it.id, data); err != nil {
				return err
			}
			moved++
		}
		if m.dryRun {
			break
		}
	}

	for {
		rows, err := m.pool.Query(ctx, `
			SELECT owner_type, owner_id, storage_path FROM avatars
			WHERE bytes IS NULL AND storage_path <> '' LIMIT $1`, m.batch)
		if err != nil {
			return fmt.Errorf("blobmigrate: select avatars back: %w", err)
		}
		type item struct {
			ownerType string
			ownerID   uuid.UUID
			key       string
		}
		var batch []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.ownerType, &it.ownerID, &it.key); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, it)
		}
		rows.Close()
		if len(batch) == 0 {
			break
		}
		for _, it := range batch {
			data, err := m.store.Get(ctx, it.key)
			if err != nil {
				return err
			}
			if m.dryRun {
				moved++
				continue
			}
			if _, err := m.pool.Exec(ctx,
				`UPDATE avatars SET bytes = $3, storage_path = '' WHERE owner_type = $1 AND owner_id = $2`,
				it.ownerType, it.ownerID, data); err != nil {
				return err
			}
			moved++
		}
		if m.dryRun {
			break
		}
	}

	m.log.Info("moved back to database", "rows", moved, "dryRun", m.dryRun)
	return nil
}
