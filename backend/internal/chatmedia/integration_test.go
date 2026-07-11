//go:build integration

package chatmedia_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/attachments"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/chatmedia"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
)

type harness struct {
	svc   *chatmedia.Service
	msgs  *messages.Service
	atts  *attachments.Service
	pool  *pgxpool.Pool
	chat  uuid.UUID
	a, b  uuid.UUID
	ctx   context.Context
	actor chatmedia.Actor
}

func setup(t *testing.T) harness {
	pool := testdb.New(t)
	ctx := context.Background()

	a := testdb.SeedUser(t, pool, "alice", 3)
	b := testdb.SeedUser(t, pool, "bob", 8)
	levels := map[uuid.UUID]int{a: 3, b: 8}
	lookup := func(_ context.Context, id uuid.UUID) (int, bool) {
		lvl, ok := levels[id]
		return lvl, ok
	}

	chatsSvc := chats.NewService(pool, chats.NewPostgresRepository(), lookup)
	chat, err := chatsSvc.OpenPrivateChat(ctx, b, chats.ActorMeta{UserID: a, RoleLevel: 3})
	if err != nil {
		t.Fatalf("open chat: %v", err)
	}

	private := func(ctx context.Context, chatID, actorID uuid.UUID) error {
		ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return chatmedia.ErrNotFound
		}
		return nil
	}
	svc := chatmedia.NewService(pool, chatmedia.NewPostgresRepository(), chatmedia.Authorizer{
		Private: private,
		Group:   func(context.Context, uuid.UUID, uuid.UUID, int) error { return chatmedia.ErrNotFound },
	})

	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	msgs := messages.NewService(pool, messages.NewPostgresRepository(), rec, messages.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(context.Context, uuid.UUID, uuid.UUID, int) error { return messages.ErrNotFound },
	})

	atts := attachments.NewService(pool, attachments.NewPostgresRepository(), attachments.Limits{
		MaxBytesLeadership: 1 << 20, MaxBytesStaff: 1 << 20, LeadershipMaxLevel: 3,
		ChunkBytes: 64 << 10, SessionTTL: time.Hour,
	})
	msgs.SetAttachments(
		func(ctx context.Context, ids []uuid.UUID, messageID, uploader uuid.UUID) error {
			return atts.Link(ctx, pool, ids, messageID, uploader)
		},
		nil,
	)

	return harness{
		svc: svc, msgs: msgs, atts: atts, pool: pool, chat: chat.ID,
		a: a, b: b, ctx: ctx, actor: chatmedia.Actor{UserID: a, RoleLevel: 3},
	}
}

// sendWithAttachment uploads bytes as the given kind and sends a message
// carrying it, returning the attachment id.
func sendWithAttachment(t *testing.T, h harness, name string, raw []byte, meta attachments.Meta) uuid.UUID {
	t.Helper()
	dto, err := h.atts.Upload(h.ctx, name, raw, h.a, 3, meta)
	if err != nil {
		t.Fatalf("upload %s: %v", name, err)
	}
	_, err = h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: h.chat, AttachmentIDs: []uuid.UUID{dto.ID},
	}, messages.ActorMeta{UserID: h.a, RoleLevel: 3})
	if err != nil {
		t.Fatalf("send %s: %v", name, err)
	}
	return dto.ID
}

var pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}

func TestTabsSplitByKind(t *testing.T) {
	h := setup(t)

	imgID := sendWithAttachment(t, h, "pic.png", append(pngMagic, make([]byte, 64)...), attachments.Meta{})
	fileID := sendWithAttachment(t, h, "doc.txt", []byte("plain text file"), attachments.Meta{})
	webm := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, make([]byte, 64)...)
	voiceID := sendWithAttachment(t, h, "note.webm", webm, attachments.Meta{
		Kind: attachments.KindVoice, DurationMs: func() *int32 { v := int32(1500); return &v }(),
	})

	media, err := h.svc.ListAttachments(h.ctx, "private", h.chat, chatmedia.TabMedia, "", 0, h.actor)
	if err != nil || len(media.Items) != 1 || media.Items[0].Attachment.ID != imgID {
		t.Fatalf("media tab: %v, %+v", err, media)
	}
	files, err := h.svc.ListAttachments(h.ctx, "private", h.chat, chatmedia.TabFiles, "", 0, h.actor)
	if err != nil || len(files.Items) != 1 || files.Items[0].Attachment.ID != fileID {
		t.Fatalf("files tab: %v, %+v", err, files)
	}
	// Voice notes belong to neither tab.
	for _, page := range []*chatmedia.MediaPage{media, files} {
		for _, it := range page.Items {
			if it.Attachment.ID == voiceID {
				t.Fatalf("voice note leaked into a tab: %+v", it)
			}
		}
	}
}

func TestLinksTab(t *testing.T) {
	h := setup(t)

	send := func(text string) {
		if _, err := h.msgs.Send(h.ctx, messages.SendInput{
			ChatType: "private", ChatID: h.chat, Text: text,
		}, messages.ActorMeta{UserID: h.a, RoleLevel: 3}); err != nil {
			t.Fatalf("send: %v", err)
		}
	}
	send("без ссылок")
	send("глянь https://example.com/doc и http://kisy.local/page.")

	links, err := h.svc.ListLinks(h.ctx, "private", h.chat, "", 0, h.actor)
	if err != nil {
		t.Fatalf("links: %v", err)
	}
	if len(links.Items) != 2 {
		t.Fatalf("links = %+v, want 2", links.Items)
	}
	if links.Items[0].URL != "https://example.com/doc" || links.Items[1].URL != "http://kisy.local/page" {
		t.Fatalf("unexpected links: %+v", links.Items)
	}

	// E2EE messages contribute nothing: the server has no text to scan.
	alg := int16(1)
	if _, err := h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: h.chat,
		Ciphertext: []byte("https://secret.example.com inside ciphertext bytes"), Alg: &alg,
	}, messages.ActorMeta{UserID: h.a, RoleLevel: 3}); err != nil {
		t.Fatalf("send e2ee: %v", err)
	}
	links2, _ := h.svc.ListLinks(h.ctx, "private", h.chat, "", 0, h.actor)
	if len(links2.Items) != 2 {
		t.Fatalf("ciphertext leaked into links tab: %+v", links2.Items)
	}
}

func TestPaginationAndDeletedExcluded(t *testing.T) {
	h := setup(t)

	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		ids = append(ids, sendWithAttachment(t, h, "p.png", append(pngMagic, byte(i)), attachments.Meta{}))
		time.Sleep(5 * time.Millisecond) // distinct created_at for stable cursors
	}

	first, err := h.svc.ListAttachments(h.ctx, "private", h.chat, chatmedia.TabMedia, "", 2, h.actor)
	if err != nil || len(first.Items) != 2 || !first.HasMore {
		t.Fatalf("page1: %v, %+v", err, first)
	}
	second, err := h.svc.ListAttachments(h.ctx, "private", h.chat, chatmedia.TabMedia, first.NextCursor, 2, h.actor)
	if err != nil || len(second.Items) != 2 || !second.HasMore {
		t.Fatalf("page2: %v, %+v", err, second)
	}
	third, err := h.svc.ListAttachments(h.ctx, "private", h.chat, chatmedia.TabMedia, second.NextCursor, 2, h.actor)
	if err != nil || len(third.Items) != 1 || third.HasMore {
		t.Fatalf("page3: %v, %+v", err, third)
	}
	// Newest-first and no duplicates across pages.
	seen := map[uuid.UUID]bool{}
	for _, p := range []*chatmedia.MediaPage{first, second, third} {
		for _, it := range p.Items {
			if seen[it.Attachment.ID] {
				t.Fatalf("duplicate across pages: %s", it.Attachment.ID)
			}
			seen[it.Attachment.ID] = true
		}
	}
	if !seen[ids[0]] || !seen[ids[4]] {
		t.Fatalf("missing items across pages")
	}
}

func TestAccessDenied(t *testing.T) {
	h := setup(t)
	sendWithAttachment(t, h, "p.png", append(pngMagic, make([]byte, 16)...), attachments.Meta{})

	stranger := chatmedia.Actor{UserID: uuid.New(), RoleLevel: 5}
	if _, err := h.svc.ListAttachments(h.ctx, "private", h.chat, chatmedia.TabMedia, "", 0, stranger); !errors.Is(err, chatmedia.ErrNotFound) {
		t.Fatalf("stranger media: want ErrNotFound (masked), got %v", err)
	}
	if _, err := h.svc.ListLinks(h.ctx, "private", h.chat, "", 0, stranger); !errors.Is(err, chatmedia.ErrNotFound) {
		t.Fatalf("stranger links: want ErrNotFound (masked), got %v", err)
	}
}
