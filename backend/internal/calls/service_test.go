package calls

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/db"
)

// --- fakes ---

type fakeStore struct {
	calls  map[uuid.UUID]CallState
	busy   map[uuid.UUID]uuid.UUID // user -> callID (mirrors the Redis busy marker)
	online map[uuid.UUID]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		calls:  map[uuid.UUID]CallState{},
		busy:   map[uuid.UUID]uuid.UUID{},
		online: map[uuid.UUID]bool{},
	}
}

func (f *fakeStore) Create(_ context.Context, s CallState) error {
	f.calls[s.ID] = s
	f.busy[s.Caller] = s.ID
	f.busy[s.Callee] = s.ID
	return nil
}
func (f *fakeStore) Get(_ context.Context, id uuid.UUID) (CallState, bool, error) {
	s, ok := f.calls[id]
	return s, ok, nil
}
func (f *fakeStore) MarkAnswered(_ context.Context, id uuid.UUID, at time.Time) error {
	s, ok := f.calls[id]
	if !ok {
		return nil
	}
	s.Phase = phaseConnected
	s.AnsweredAt = &at
	f.calls[id] = s
	return nil
}
func (f *fakeStore) Delete(_ context.Context, id uuid.UUID) error {
	if s, ok := f.calls[id]; ok {
		delete(f.busy, s.Caller)
		delete(f.busy, s.Callee)
	}
	delete(f.calls, id)
	return nil
}
func (f *fakeStore) UserBusy(_ context.Context, id uuid.UUID) (bool, error) {
	_, ok := f.busy[id]
	return ok, nil
}
func (f *fakeStore) IsOnline(_ context.Context, id uuid.UUID) (bool, error) { return f.online[id], nil }
func (f *fakeStore) CallIDForUser(_ context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	cid, ok := f.busy[id]
	return cid, ok, nil
}
func (f *fakeStore) ClearUserBusy(_ context.Context, id uuid.UUID) error {
	delete(f.busy, id)
	return nil
}

type pubEvent struct {
	method string
	to     uuid.UUID
	callID uuid.UUID
}

type fakePublisher struct{ events []pubEvent }

func (p *fakePublisher) Incoming(to, callID, _ uuid.UUID, _ string, _ *string, _ uuid.UUID, _ string) {
	p.events = append(p.events, pubEvent{"incoming", to, callID})
}
func (p *fakePublisher) Answered(to, callID uuid.UUID, _ string) {
	p.events = append(p.events, pubEvent{"answered", to, callID})
}
func (p *fakePublisher) ICE(to, callID, _ uuid.UUID, _ json.RawMessage) {
	p.events = append(p.events, pubEvent{"ice", to, callID})
}
func (p *fakePublisher) Rejected(to, callID uuid.UUID) {
	p.events = append(p.events, pubEvent{"rejected", to, callID})
}
func (p *fakePublisher) Canceled(to, callID uuid.UUID) {
	p.events = append(p.events, pubEvent{"canceled", to, callID})
}
func (p *fakePublisher) Ended(to, callID uuid.UUID, _ string) {
	p.events = append(p.events, pubEvent{"ended", to, callID})
}
func (p *fakePublisher) Busy(to, callID uuid.UUID) {
	p.events = append(p.events, pubEvent{"busy", to, callID})
}
func (p *fakePublisher) Timeout(to, callID uuid.UUID) {
	p.events = append(p.events, pubEvent{"timeout", to, callID})
}
func (p *fakePublisher) has(method string, to uuid.UUID) bool {
	for _, e := range p.events {
		if e.method == method && e.to == to {
			return true
		}
	}
	return false
}
func (p *fakePublisher) count(method string) int {
	n := 0
	for _, e := range p.events {
		if e.method == method {
			n++
		}
	}
	return n
}

type finalRec struct {
	status   string
	duration int
	answered *time.Time
	ended    *time.Time
}

type fakeRepo struct {
	created map[uuid.UUID]CallLog
	final   map[uuid.UUID]finalRec
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{created: map[uuid.UUID]CallLog{}, final: map[uuid.UUID]finalRec{}}
}
func (r *fakeRepo) Create(_ context.Context, _ db.DBTX, log CallLog) error {
	r.created[log.ID] = log
	return nil
}
func (r *fakeRepo) Finalize(_ context.Context, _ db.DBTX, id uuid.UUID, status string, answered, ended *time.Time, duration int) error {
	r.final[id] = finalRec{status: status, duration: duration, answered: answered, ended: ended}
	return nil
}
func (r *fakeRepo) ListForUser(_ context.Context, _ db.DBTX, _ uuid.UUID, _, _ int) ([]CallLogRow, error) {
	return nil, nil
}

type fakeAccess struct {
	members map[uuid.UUID]map[uuid.UUID]bool
}

func (a fakeAccess) IsParticipant(_ context.Context, chatID, userID uuid.UUID) (bool, error) {
	return a.members[chatID][userID], nil
}
func (a fakeAccess) ParticipantIDs(_ context.Context, chatID uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for id := range a.members[chatID] {
		out = append(out, id)
	}
	return out, nil
}

type noopAudit struct{}

func (noopAudit) Record(_ context.Context, _ db.DBTX, _ audit.Event) error { return nil }

// --- harness ---

type harness struct {
	svc    *Service
	store  *fakeStore
	pub    *fakePublisher
	repo   *fakeRepo
	chatID uuid.UUID
	alice  uuid.UUID
	bob    uuid.UUID
	clock  time.Time
	fired  []func() // captured ring-timeout callbacks
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{
		store:  newFakeStore(),
		pub:    &fakePublisher{},
		repo:   newFakeRepo(),
		chatID: uuid.New(),
		alice:  uuid.New(),
		bob:    uuid.New(),
		clock:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	h.store.online[h.alice] = true
	h.store.online[h.bob] = true
	access := fakeAccess{members: map[uuid.UUID]map[uuid.UUID]bool{
		h.chatID: {h.alice: true, h.bob: true},
	}}
	h.svc = NewService(nil, h.repo, h.store, access, noopAudit{},
		ICESettings{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.svc.SetPublisher(h.pub)
	h.svc.now = func() time.Time { return h.clock }
	h.svc.afterFunc = func(_ time.Duration, f func()) { h.fired = append(h.fired, f) }
	return h
}

func (h *harness) actor(id uuid.UUID) Actor { return Actor{UserID: id} }

func (h *harness) invite(t *testing.T, from, to uuid.UUID, callID uuid.UUID) error {
	t.Helper()
	data, _ := json.Marshal(invitePayload{CallID: callID, ToUserID: to, ChatID: h.chatID, SDP: "offer"})
	return h.svc.HandleSignal(context.Background(), h.actor(from), SignalInvite, data)
}

func (h *harness) signal(t *testing.T, actor uuid.UUID, typ string, payload any) error {
	t.Helper()
	data, _ := json.Marshal(payload)
	return h.svc.HandleSignal(context.Background(), h.actor(actor), typ, data)
}

// --- tests ---

func TestInviteRingsCallee(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if !h.pub.has("incoming", h.bob) {
		t.Fatal("expected call.incoming to callee")
	}
	if _, ok := h.store.calls[callID]; !ok {
		t.Fatal("expected live call state")
	}
	if _, ok := h.repo.created[h.store.calls[callID].LogID]; !ok {
		t.Fatal("expected a call_logs row")
	}
	if len(h.fired) != 1 {
		t.Fatalf("expected a scheduled ring-timeout, got %d", len(h.fired))
	}
}

func TestInviteDeniedForNonParticipant(t *testing.T) {
	h := newHarness(t)
	stranger := uuid.New()
	h.store.online[stranger] = true
	// Stranger is not a member of the chat.
	data, _ := json.Marshal(invitePayload{CallID: uuid.New(), ToUserID: stranger, ChatID: h.chatID, SDP: "offer"})
	err := h.svc.HandleSignal(context.Background(), h.actor(h.alice), SignalInvite, data)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
	if len(h.pub.events) != 0 {
		t.Fatal("no events should be published for a denied invite")
	}
}

// markOnCall puts the user on a genuine in-flight call (busy marker plus the
// live state it points at), as opposed to an orphaned marker.
func (h *harness) markOnCall(user uuid.UUID) uuid.UUID {
	cid := uuid.New()
	h.store.calls[cid] = CallState{
		ID: cid, Caller: user, Callee: uuid.New(), ChatID: h.chatID,
		Phase: phaseRinging, StartedAt: h.clock,
	}
	h.store.busy[user] = cid
	return cid
}

func TestInviteBusyCallee(t *testing.T) {
	h := newHarness(t)
	h.markOnCall(h.bob) // Bob already on another call
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if !h.pub.has("busy", h.alice) {
		t.Fatal("expected call.busy to caller")
	}
	if _, ok := h.store.calls[callID]; ok {
		t.Fatal("no live state should be created for a busy callee")
	}
}

// A busy marker can outlive its call when the process dies mid-conversation
// (crash/redeploy), so HandleDisconnect never runs. Such an orphan must not
// block calling: the caller's own stale marker used to end every new attempt
// instantly with "busy", and a callee's stale marker made them permanently
// unreachable.
func TestInviteReapsOrphanBusyMarkerOnCaller(t *testing.T) {
	h := newHarness(t)
	h.store.busy[h.alice] = uuid.New() // marker pointing at a call that is gone
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if h.pub.has("ended", h.alice) {
		t.Fatal("an orphaned marker must not end the call as busy")
	}
	if !h.pub.has("incoming", h.bob) {
		t.Fatal("expected the call to ring the callee")
	}
}

func TestInviteReapsOrphanBusyMarkerOnCallee(t *testing.T) {
	h := newHarness(t)
	h.store.busy[h.bob] = uuid.New() // marker pointing at a call that is gone
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if h.pub.has("busy", h.alice) {
		t.Fatal("an orphaned marker must not report the callee busy")
	}
	if !h.pub.has("incoming", h.bob) {
		t.Fatal("expected the call to ring the callee")
	}
}

func TestInviteOfflineCallee(t *testing.T) {
	h := newHarness(t)
	h.store.online[h.bob] = false
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if !h.pub.has("timeout", h.alice) {
		t.Fatal("expected call.timeout to caller for offline callee")
	}
}

func TestGlareSecondInviteRejected(t *testing.T) {
	h := newHarness(t)
	// Alice invites Bob first; both become busy.
	if err := h.invite(t, h.alice, h.bob, uuid.New()); err != nil {
		t.Fatalf("first invite: %v", err)
	}
	// Bob simultaneously invites Alice: Bob is now busy → his attempt ends
	// gracefully (call.ended to Bob), and Alice is not rung a second time.
	glareID := uuid.New()
	if err := h.invite(t, h.bob, h.alice, glareID); err != nil {
		t.Fatalf("glare invite should not error, got %v", err)
	}
	if !h.pub.has("ended", h.bob) {
		t.Fatal("busy caller should receive call.ended for the glare attempt")
	}
	if h.pub.has("incoming", h.alice) {
		t.Fatal("Alice must not be rung by the glare invite")
	}
}

func TestAnswerOnlyByCallee(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	// Caller cannot "answer" their own call.
	if err := h.signal(t, h.alice, SignalAnswer, answerPayload{CallID: callID, SDP: "answer"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("caller answer: got %v, want ErrForbidden", err)
	}
	// Callee answers.
	if err := h.signal(t, h.bob, SignalAnswer, answerPayload{CallID: callID, SDP: "answer"}); err != nil {
		t.Fatalf("callee answer: %v", err)
	}
	if !h.pub.has("answered", h.alice) {
		t.Fatal("expected call.answered to caller")
	}
	if h.store.calls[callID].Phase != phaseConnected {
		t.Fatal("expected connected phase after answer")
	}
}

func TestICERelayAndAuthorization(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	// Alice's ICE relays to Bob.
	if err := h.signal(t, h.alice, SignalICE, icePayload{CallID: callID, Candidate: json.RawMessage(`{"c":1}`)}); err != nil {
		t.Fatalf("ice: %v", err)
	}
	if !h.pub.has("ice", h.bob) {
		t.Fatal("expected ICE relayed to Bob")
	}
	// An outsider cannot inject ICE into the call.
	if err := h.signal(t, uuid.New(), SignalICE, icePayload{CallID: callID, Candidate: json.RawMessage(`{"c":2}`)}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider ice: got %v, want ErrForbidden", err)
	}
}

func TestRejectFinalizesRejected(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	logID := h.store.calls[callID].LogID
	if err := h.signal(t, h.bob, SignalReject, refPayload{CallID: callID}); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if !h.pub.has("rejected", h.alice) {
		t.Fatal("expected call.rejected to caller")
	}
	if h.repo.final[logID].status != StatusRejected {
		t.Fatalf("log status = %q, want rejected", h.repo.final[logID].status)
	}
	if _, ok := h.store.calls[callID]; ok {
		t.Fatal("call state should be cleared after reject")
	}
}

func TestHangupAfterAnswerRecordsDuration(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	if err := h.signal(t, h.bob, SignalAnswer, answerPayload{CallID: callID, SDP: "answer"}); err != nil {
		t.Fatal(err)
	}
	logID := h.store.calls[callID].LogID
	// 90 seconds pass, then Alice hangs up.
	h.clock = h.clock.Add(90 * time.Second)
	if err := h.signal(t, h.alice, SignalHangup, refPayload{CallID: callID}); err != nil {
		t.Fatalf("hangup: %v", err)
	}
	if !h.pub.has("ended", h.bob) {
		t.Fatal("expected call.ended to the other party")
	}
	fr := h.repo.final[logID]
	if fr.status != StatusCompleted {
		t.Fatalf("status = %q, want completed", fr.status)
	}
	if fr.duration != 90 {
		t.Fatalf("duration = %d, want 90", fr.duration)
	}
}

func TestRingTimeoutMarksMissed(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	logID := h.store.calls[callID].LogID
	// Fire the captured ring-timeout callback (no answer arrived).
	if len(h.fired) != 1 {
		t.Fatalf("expected 1 scheduled timeout, got %d", len(h.fired))
	}
	h.fired[0]()

	if h.pub.count("timeout") != 2 {
		t.Fatalf("expected timeout to both parties, got %d", h.pub.count("timeout"))
	}
	if h.repo.final[logID].status != StatusMissed {
		t.Fatalf("status = %q, want missed", h.repo.final[logID].status)
	}
	if _, ok := h.store.calls[callID]; ok {
		t.Fatal("call state should be cleared after timeout")
	}
}

func TestRingTimeoutNoopAfterAnswer(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	if err := h.signal(t, h.bob, SignalAnswer, answerPayload{CallID: callID, SDP: "answer"}); err != nil {
		t.Fatal(err)
	}
	// A late ring-timeout must not disturb an already-answered call.
	h.fired[0]()
	if h.pub.count("timeout") != 0 {
		t.Fatal("answered call must not time out")
	}
	if _, ok := h.store.calls[callID]; !ok {
		t.Fatal("answered call state must survive a stale timeout")
	}
}

func TestDisconnectEndsRingingCall(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	logID := h.store.calls[callID].LogID
	// Caller's last connection drops while the call is still ringing.
	h.svc.HandleDisconnect(context.Background(), h.alice)

	if !h.pub.has("ended", h.bob) {
		t.Fatal("callee should be told the call ended when caller disconnects")
	}
	if _, ok := h.store.calls[callID]; ok {
		t.Fatal("call state should be cleared on disconnect")
	}
	if busy, _ := h.store.UserBusy(context.Background(), h.alice); busy {
		t.Fatal("caller busy marker should be cleared")
	}
	if h.repo.final[logID].status != StatusCanceled {
		t.Fatalf("status = %q, want canceled", h.repo.final[logID].status)
	}
	// A fresh call can now be placed (no stale busy marker blocking it).
	if err := h.invite(t, h.alice, h.bob, uuid.New()); err != nil {
		t.Fatalf("call after disconnect should succeed, got %v", err)
	}
}

func TestCalleeDisconnectDoesNotRefuseARingingCall(t *testing.T) {
	// The phone was swept out of memory and woken by a push. The socket it had
	// before dies right about now — and a dying socket is not a refusal. This
	// was answering "Ответить" with a rejected call: the app opened, the old
	// connection was reaped, and the caller heard "Звонок отклонён".
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatal(err)
	}
	logID := h.store.calls[callID].LogID

	h.svc.HandleDisconnect(context.Background(), h.bob)

	if _, ok := h.store.calls[callID]; !ok {
		t.Fatal("a ringing call must survive the callee's socket dropping")
	}
	if h.pub.has("ended", h.alice) {
		t.Fatal("caller must not be told the call ended")
	}
	if _, finalized := h.repo.final[logID]; finalized {
		t.Fatalf("call must not be finalized, got %q", h.repo.final[logID].status)
	}

	// And it still ends by itself when nobody ever picks up.
	h.svc.onRingTimeout(context.Background(), callID)
	if h.repo.final[logID].status != StatusMissed {
		t.Fatalf("status = %q, want missed", h.repo.final[logID].status)
	}
}

func TestDisconnectReapsOrphanBusyMarker(t *testing.T) {
	h := newHarness(t)
	// Simulate an orphaned marker: busy set but no live call state.
	h.store.busy[h.alice] = uuid.New()
	h.svc.HandleDisconnect(context.Background(), h.alice)
	if busy, _ := h.store.UserBusy(context.Background(), h.alice); busy {
		t.Fatal("orphaned busy marker should be reaped on disconnect")
	}
}

func TestUnknownSignalRejected(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.HandleSignal(context.Background(), h.actor(h.alice), "call.bogus", json.RawMessage(`{}`)); !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v, want ErrValidation", err)
	}
}

// fakePusher stands in for Firebase.
type fakePusher struct {
	devices map[uuid.UUID]bool
	sent    []map[string]string
	to      []uuid.UUID
}

func (p *fakePusher) HasDevices(_ context.Context, userID uuid.UUID) bool { return p.devices[userID] }
func (p *fakePusher) SendData(_ context.Context, userID uuid.UUID, data map[string]string, _ time.Duration) bool {
	p.to = append(p.to, userID)
	p.sent = append(p.sent, data)
	return p.devices[userID]
}
func (p *fakePusher) last(kind string) map[string]string {
	for i := len(p.sent) - 1; i >= 0; i-- {
		if p.sent[i]["type"] == kind {
			return p.sent[i]
		}
	}
	return nil
}

// The bug: a callee with no live WebSocket was written off as missed on the
// spot, so a phone with the app closed — the normal state of a phone — could
// never be called at all.
func TestInviteRingsOfflineCalleeThroughPush(t *testing.T) {
	h := newHarness(t)
	pusher := &fakePusher{devices: map[uuid.UUID]bool{h.bob: true}}
	h.svc.SetPusher(pusher)
	h.store.online[h.bob] = false

	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}

	if h.pub.has("timeout", h.alice) {
		t.Fatal("caller was told the call timed out immediately; the phone never got a chance to ring")
	}
	invite := pusher.last("call_invite")
	if invite == nil {
		t.Fatal("no call_invite push was sent to the offline callee")
	}
	if invite["callId"] != callID.String() {
		t.Errorf("callId = %q, want %q", invite["callId"], callID)
	}
	if invite["callerId"] != h.alice.String() {
		t.Errorf("callerId = %q", invite["callerId"])
	}
	// The call must exist, otherwise answering the push would find nothing.
	if _, ok, _ := h.store.Get(context.Background(), callID); !ok {
		t.Fatal("call state was not created for a pushed invite")
	}
}

// Nothing to ring: the caller should learn that at once rather than listen to
// a ringtone for the full timeout.
func TestInviteFailsFastWhenCalleeHasNoDevices(t *testing.T) {
	h := newHarness(t)
	h.svc.SetPusher(&fakePusher{devices: map[uuid.UUID]bool{}})
	h.store.online[h.bob] = false

	if err := h.invite(t, h.alice, h.bob, uuid.New()); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if !h.pub.has("timeout", h.alice) {
		t.Fatal("caller was left ringing for a callee that cannot be reached")
	}
}

// A notification left ringing for a call the caller already dropped is the
// worst outcome of the whole feature.
func TestCancelAndTimeoutSilenceThePhone(t *testing.T) {
	h := newHarness(t)
	pusher := &fakePusher{devices: map[uuid.UUID]bool{h.bob: true}}
	h.svc.SetPusher(pusher)
	h.store.online[h.bob] = false

	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := h.signal(t, h.alice, SignalCancel, refPayload{CallID: callID}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := pusher.last("call_cancel"); got == nil || got["callId"] != callID.String() {
		t.Fatalf("cancel push missing or wrong: %v", got)
	}

	// And the same when nobody picks up.
	h2 := newHarness(t)
	pusher2 := &fakePusher{devices: map[uuid.UUID]bool{h2.bob: true}}
	h2.svc.SetPusher(pusher2)
	h2.store.online[h2.bob] = false
	callID2 := uuid.New()
	if err := h2.invite(t, h2.alice, h2.bob, callID2); err != nil {
		t.Fatalf("invite: %v", err)
	}
	for _, f := range h2.fired {
		f()
	}
	if got := pusher2.last("call_cancel"); got == nil || got["callId"] != callID2.String() {
		t.Fatalf("ring timeout left the phone ringing: %v", got)
	}
}

// A phone woken by a push has no socket history: the invite frame it would
// have received was published while it was asleep and is gone. It has to be
// able to ask what is ringing — offer included, or it cannot answer.
func TestPendingInviteLetsAWokenPhoneAnswer(t *testing.T) {
	h := newHarness(t)
	h.svc.SetPusher(&fakePusher{devices: map[uuid.UUID]bool{h.bob: true}})
	h.store.online[h.bob] = false

	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}

	st, ok := h.svc.PendingInvite(context.Background(), h.actor(h.bob))
	if !ok {
		t.Fatal("callee cannot discover the call that is ringing for them")
	}
	if st.Offer != "offer" {
		t.Errorf("offer = %q; without it the callee has nothing to answer with", st.Offer)
	}
	if st.ID != callID || st.Caller != h.alice {
		t.Errorf("wrong call returned: %+v", st)
	}

	// The caller must not see it as something to answer.
	if _, ok := h.svc.PendingInvite(context.Background(), h.actor(h.alice)); ok {
		t.Error("the caller was offered their own call to answer")
	}
}

// Declining from a notification happens before the socket is up.
func TestRejectOverRESTEndsTheCall(t *testing.T) {
	h := newHarness(t)
	callID := uuid.New()
	if err := h.invite(t, h.alice, h.bob, callID); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := h.svc.Reject(context.Background(), h.actor(h.bob), callID); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if !h.pub.has("rejected", h.alice) {
		t.Fatal("caller was never told the call was declined")
	}
	// And a stranger cannot decline someone else's call.
	h2 := newHarness(t)
	id2 := uuid.New()
	if err := h2.invite(t, h2.alice, h2.bob, id2); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := h2.svc.Reject(context.Background(), Actor{UserID: uuid.New()}, id2); err == nil {
		t.Fatal("an unrelated user was allowed to decline the call")
	}
}
