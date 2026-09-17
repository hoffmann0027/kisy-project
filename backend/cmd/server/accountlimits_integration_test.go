//go:build integration

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	goredis "github.com/redis/go-redis/v9"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/ratelimit"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/ws"
)

// Per-account limits (audit A-23), through the real services and handlers:
// every refusal is a 429 carrying Retry-After, the REST and WebSocket send
// paths draw on one budget, and basic and invited accounts get their own.

func testRedis(t *testing.T) *goredis.Client {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set; skipping integration test")
	}
	opts, err := goredis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	rdb := goredis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// tightPolicy: small budgets so the tests reach them, and different ones for
// the two kinds of account.
func tightPolicy() ratelimit.AccountPolicy {
	tier := func(basic, invited int, window time.Duration) ratelimit.Tiered {
		return ratelimit.Tiered{Basic: ratelimit.Rule{Max: basic, Window: window}, Invited: ratelimit.Rule{Max: invited, Window: window}}
	}
	return ratelimit.AccountPolicy{
		ratelimit.ScopeMessages:    tier(3, 5, time.Minute),
		ratelimit.ScopeNewChats:    tier(1, 3, time.Hour),
		ratelimit.ScopeSearch:      tier(2, 4, time.Minute),
		ratelimit.ScopeLinkPreview: tier(2, 4, time.Minute),
		ratelimit.ScopeUploads:     tier(2, 4, time.Hour),
	}
}

func assertLimited(t *testing.T, label string, res *http.Response) {
	t.Helper()
	if res.StatusCode != http.StatusTooManyRequests {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("%s: status %d, want 429 (%s)", label, res.StatusCode, body)
	}
	if secs, err := strconv.Atoi(res.Header.Get("Retry-After")); err != nil || secs < 1 {
		t.Fatalf("%s: Retry-After = %q", label, res.Header.Get("Retry-After"))
	}
}

type limitsFixture struct {
	ctx      context.Context
	msgs     *messages.Service
	chats    *chats.Service
	basic    uuid.UUID
	basic2   uuid.UUID
	basic3   uuid.UUID
	chat     *chats.PrivateChat
	levels   map[uuid.UUID]int
	accounts *ratelimit.Accounts
}

func newLimitsFixture(t *testing.T) limitsFixture {
	t.Helper()
	pool := testdb.New(t)
	rdb := testRedis(t)
	ctx := context.Background()
	f := limitsFixture{ctx: ctx, levels: map[uuid.UUID]int{}}
	suffix := uuid.NewString()[:8]
	f.basic = testdb.SeedUser(t, pool, "basic_a_"+suffix, 0)
	f.basic2 = testdb.SeedUser(t, pool, "basic_b_"+suffix, 0)
	f.basic3 = testdb.SeedUser(t, pool, "basic_c_"+suffix, 0)
	for _, id := range []uuid.UUID{f.basic, f.basic2, f.basic3} {
		f.levels[id] = 0
	}
	f.chats = chats.NewService(pool, chats.NewPostgresRepository(), func(_ context.Context, id uuid.UUID) (int, bool) {
		lvl, ok := f.levels[id]
		return lvl, ok
	})
	f.msgs = messages.NewService(pool, messages.NewPostgresRepository(), audit.NewPostgresRecorder(quiet()), messages.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			if ok, err := f.chats.IsParticipant(ctx, chatID, actorID); err != nil || !ok {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(context.Context, uuid.UUID, uuid.UUID, int) error { return messages.ErrNotFound },
	})
	// The chat the message tests write into exists before the limits are on.
	chat, err := f.chats.OpenPrivateChat(ctx, f.basic2, chats.ActorMeta{UserID: f.basic})
	if err != nil {
		t.Fatal(err)
	}
	f.chat = chat
	f.accounts = ratelimit.NewAccounts(ratelimit.NewLimiter(rdb, quiet()), tightPolicy())
	wireAccountLimits(f.accounts, f.msgs, f.chats)
	return f
}

func ciphertextBody(chatID uuid.UUID) string {
	return `{"chatType":"private","chatId":"` + chatID.String() + `","ciphertext":"Y2lwaGVy","alg":1,"epoch":1}`
}

func TestMessagesShareOneBudgetAcrossRESTAndWebSocket(t *testing.T) {
	f := newLimitsFixture(t)

	// REST.
	h := messages.NewHandler(f.msgs, func(*http.Request) (messages.ActorMeta, bool) {
		return messages.ActorMeta{UserID: f.basic}, true
	})
	r := chi.NewRouter()
	h.Routes(r)
	api := httptest.NewServer(r)
	defer api.Close()
	post := func() *http.Response {
		res, err := http.Post(api.URL+"/messages", "application/json", strings.NewReader(ciphertextBody(f.chat.ID)))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		return res
	}
	for i := 0; i < 2; i++ {
		if res := post(); res.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("REST send %d: %d %s", i+1, res.StatusCode, body)
		}
	}

	// WebSocket: the third message of the budget goes through, the fourth is
	// refused — with the same retry hint.
	hub := ws.NewHub(quiet(), testRedis(t), func(context.Context, string, uuid.UUID) ([]uuid.UUID, error) { return nil, nil })
	hub.SetHandlers(f.msgs, nil, nil)
	wsSrv := httptest.NewServer(ws.NewHandler(hub, func(*http.Request) (ws.Authenticated, bool) {
		return ws.Authenticated{UserID: f.basic, SessionID: uuid.New()}, true
	}, ""))
	defer wsSrv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(wsSrv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	send := func() {
		frame := `{"type":"message.send","data":` + ciphertextBody(f.chat.ID) + `}`
		if err := conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
			t.Fatal(err)
		}
	}
	send()
	send()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var refusal struct {
		Event string `json:"event"`
		Data  struct {
			Code       string `json:"code"`
			RetryAfter int    `json:"retryAfter"`
		} `json:"data"`
	}
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("no refusal over the socket: %v", err)
		}
		if json.Unmarshal(raw, &refusal) == nil && refusal.Event == ws.EventError {
			break
		}
	}
	if refusal.Data.Code != "RATE_LIMITED" || refusal.Data.RetryAfter < 1 {
		t.Fatalf("socket refusal = %+v", refusal.Data)
	}

	// And REST is out of budget too.
	assertLimited(t, "REST after the socket spent the budget", post())
}

func TestInvitedAccountsGetTheirOwnBudget(t *testing.T) {
	f := newLimitsFixture(t)
	invited := uuid.New()
	for i := 0; i < 5; i++ {
		if err := f.accounts.Check(f.ctx, ratelimit.ScopeMessages, invited, 7); err != nil {
			t.Fatalf("invited hit %d: %v", i+1, err)
		}
	}
	if _, limited := ratelimit.AsLimited(f.accounts.Check(f.ctx, ratelimit.ScopeMessages, invited, 7)); !limited {
		t.Fatal("invited account over its budget was not refused")
	}
	basic := uuid.New()
	for i := 0; i < 3; i++ {
		_ = f.accounts.Check(f.ctx, ratelimit.ScopeMessages, basic, 0)
	}
	if _, limited := ratelimit.AsLimited(f.accounts.Check(f.ctx, ratelimit.ScopeMessages, basic, 0)); !limited {
		t.Fatal("basic account over its budget was not refused")
	}
}

func TestOnlyNewConversationsSpendTheNewChatBudget(t *testing.T) {
	f := newLimitsFixture(t)
	h := chats.NewHandler(f.chats, func(*http.Request) (chats.ActorMeta, bool) {
		return chats.ActorMeta{UserID: f.basic2}, true
	})
	r := chi.NewRouter()
	h.Routes(r)
	srv := httptest.NewServer(r)
	defer srv.Close()
	open := func(target uuid.UUID) *http.Response {
		res, err := http.Post(srv.URL+"/", "application/json", strings.NewReader(`{"userId":"`+target.String()+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		return res
	}
	// Reopening the chat that exists costs nothing, however often.
	for i := 0; i < 3; i++ {
		if res := open(f.basic); res.StatusCode != http.StatusOK {
			t.Fatalf("reopen %d: %d", i+1, res.StatusCode)
		}
	}
	// A basic account's budget here is one new conversation.
	if res := open(f.basic3); res.StatusCode != http.StatusOK {
		t.Fatalf("first new chat: %d", res.StatusCode)
	}
	extra := uuid.New()
	f.levels[extra] = 0
	assertLimited(t, "second new chat", open(extra))
}

func TestRequestScopesAreLimitedPerAccount(t *testing.T) {
	rdb := testRedis(t)
	accounts := ratelimit.NewAccounts(ratelimit.NewLimiter(rdb, quiet()), tightPolicy())
	r := chi.NewRouter()
	user := uuid.New()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := auth.ContextWithClaims(req.Context(), &token.AccessClaims{UserID: user, Kind: "basic"})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Use(accounts.Middleware(claimsIdentity, accountLimitedScope))
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	r.Get("/api/v1/search/", ok)
	r.Post("/api/v1/link-preview", ok)
	r.Post("/api/v1/groups/", ok)
	r.Post("/api/v1/attachments", ok)
	r.Post("/api/v1/attachments/init", ok)
	r.Put("/api/v1/attachments/{id}/chunk", ok)
	r.Post("/api/v1/messages/read", ok)
	srv := httptest.NewServer(r)
	defer srv.Close()

	do := func(method, path string) *http.Response {
		req, _ := http.NewRequest(method, srv.URL+path, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		return res
	}
	// Basic budgets: search 2/min, previews 2/min, uploads 2/h.
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/search/"},
		{http.MethodPost, "/api/v1/link-preview"},
	} {
		for i := 0; i < 2; i++ {
			if res := do(c.method, c.path); res.StatusCode != http.StatusOK {
				t.Fatalf("%s %d: %d", c.path, i+1, res.StatusCode)
			}
		}
		assertLimited(t, c.path, do(c.method, c.path))
	}
	// Chunks of an upload are not uploads.
	for i := 0; i < 10; i++ {
		if res := do(http.MethodPut, "/api/v1/attachments/abc/chunk"); res.StatusCode != http.StatusOK {
			t.Fatalf("chunk %d: %d", i+1, res.StatusCode)
		}
	}
	do(http.MethodPost, "/api/v1/attachments/init")
	do(http.MethodPost, "/api/v1/attachments")
	assertLimited(t, "third upload", do(http.MethodPost, "/api/v1/attachments"))
	// A new group spends the new-chat budget (basic: 1/h).
	do(http.MethodPost, "/api/v1/groups/")
	assertLimited(t, "second new group", do(http.MethodPost, "/api/v1/groups/"))
	// Unmatched routes are untouched.
	for i := 0; i < 10; i++ {
		if res := do(http.MethodPost, "/api/v1/messages/read"); res.StatusCode != http.StatusOK {
			t.Fatalf("unlimited route %d: %d", i+1, res.StatusCode)
		}
	}
}
