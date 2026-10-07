//go:build integration

package auth_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/consent"
	"kisy-backend/internal/platform/turnstile"
	"kisy-backend/internal/users"
)

// Google Play's policy for apps with user-generated content: the rules that
// define objectionable content must be accepted before a person can publish
// anything. Until now KISY had no rules at all and asked for nothing — an
// account was a login, a name and a password.
//
// These tests pin that consent is a fact the SERVER keeps, not a screen the
// client may or may not show: no account is created without it, an existing
// account is asked, and the evidence cannot be rewritten afterwards.

type registerReply struct {
	status int
	code   string
	user   map[string]any
}

func registerWith(t *testing.T, srvURL string, extra map[string]any) registerReply {
	t.Helper()
	body := map[string]any{
		"username": "consent_user", "displayName": "Согласный Человек", "password": "consent-pass-12",
		"turnstileToken": "human-token",
	}
	for k, v := range extra {
		body[k] = v
	}
	raw, _ := json.Marshal(body)
	res, err := http.Post(srvURL+"/api/v1/auth/register", "application/json", strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var env struct {
		Data struct {
			User map[string]any `json:"user"`
		} `json:"data"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&env)
	reply := registerReply{status: res.StatusCode, user: env.Data.User}
	if env.Error != nil {
		reply.code = env.Error.Code
	}
	return reply
}

func TestNoAccountIsCreatedWithoutConsent(t *testing.T) {
	cf := fakeCloudflare(t)
	e, srv := captchaServer(t, turnstile.NewClient("cf-secret", cf.URL, time.Second, nil))

	cases := map[string]map[string]any{
		"nothing ticked":          {},
		"only the privacy policy": {"privacyVersion": consent.PrivacyVersion},
		"only the rules":          {"rulesVersion": consent.RulesVersion},
		"an older text":           {"privacyVersion": "2020-01-01", "rulesVersion": consent.RulesVersion},
	}
	for name, extra := range cases {
		got := registerWith(t, srv.URL, extra)
		if got.status != http.StatusBadRequest || got.code != "CONSENT_REQUIRED" {
			t.Fatalf("%s: status %d code %q, want 400 CONSENT_REQUIRED", name, got.status, got.code)
		}
		if accountExists(t, e) {
			t.Fatalf("%s: an account was created without consent", name)
		}
	}

	// The service refuses too: the HTTP check is not the only gate.
	_, err := e.svc.Register(t.Context(), "", "service_user", "Сервисный Человек", "consent-pass-12",
		consent.Acceptance{}, testMeta)
	if !errors.Is(err, consent.ErrNotAccepted) {
		t.Fatalf("service registered without consent: %v", err)
	}
}

func TestSignUpRecordsConsentWithTheAccount(t *testing.T) {
	cf := fakeCloudflare(t)
	e, srv := captchaServer(t, turnstile.NewClient("cf-secret", cf.URL, time.Second, nil))

	got := registerWith(t, srv.URL, map[string]any{
		"privacyVersion": consent.PrivacyVersion, "rulesVersion": consent.RulesVersion,
	})
	if got.status != http.StatusCreated {
		t.Fatalf("sign-up with consent: status %d code %q", got.status, got.code)
	}
	if _, asked := got.user["consentRequired"]; asked {
		t.Fatal("a fresh account is asked to accept what it has just accepted")
	}

	var privacy, rules string
	if err := e.pool.QueryRow(t.Context(), `
		SELECT privacy_version, rules_version FROM users WHERE username = 'consent_user'`).Scan(&privacy, &rules); err != nil {
		t.Fatal(err)
	}
	if privacy != consent.PrivacyVersion || rules != consent.RulesVersion {
		t.Fatalf("stored versions %q / %q", privacy, rules)
	}

	var evidence int
	if err := e.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM legal_acceptances a JOIN users u ON u.id = a.user_id
		WHERE u.username = 'consent_user'`).Scan(&evidence); err != nil {
		t.Fatal(err)
	}
	if evidence != 2 {
		t.Fatalf("evidence rows = %d, want 2 (privacy + rules)", evidence)
	}
}

func TestExistingAccountIsAskedAndTheEvidenceIsKept(t *testing.T) {
	e := setup(t)
	u := e.createUser(t, "old_timer", "old-timer-pass-1", 5)
	svc := users.NewService(e.pool, e.users, audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil))))

	// An account from before the rules existed has to accept them.
	before, err := e.users.GetByID(t.Context(), e.pool, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ToSelfDTO().ConsentRequired {
		t.Fatal("an account that never accepted anything is not asked to")
	}
	// ...and nobody else learns that about it.
	if before.ToDTO().ConsentRequired {
		t.Fatal("whether someone accepted the rules leaks into their public profile")
	}

	// A stale bundle showing last month's rules cannot accept this month's.
	if _, err := svc.AcceptConsent(t.Context(), u.ID,
		consent.Acceptance{PrivacyVersion: consent.PrivacyVersion, RulesVersion: "2020-01-01"}, users.ActorMeta{}); !errors.Is(err, consent.ErrNotAccepted) {
		t.Fatalf("an outdated acceptance was recorded: %v", err)
	}

	after, err := svc.AcceptConsent(t.Context(), u.ID, consent.Current(), users.ActorMeta{IPHash: "ip-digest"})
	if err != nil {
		t.Fatal(err)
	}
	if after.ToSelfDTO().ConsentRequired {
		t.Fatal("still asked after accepting")
	}

	// The evidence is append-only: the trigger refuses to rewrite it.
	if _, err := e.pool.Exec(t.Context(), `UPDATE legal_acceptances SET version = 'forged' WHERE user_id = $1`, u.ID); err == nil {
		t.Fatal("the evidence of consent could be rewritten")
	}

	var ipHash string
	if err := e.pool.QueryRow(t.Context(), `
		SELECT ip_hash FROM legal_acceptances WHERE user_id = $1 AND document = 'rules'`, u.ID).Scan(&ipHash); err != nil {
		t.Fatal(err)
	}
	if ipHash != "ip-digest" {
		t.Fatalf("evidence ip_hash = %q", ipHash)
	}
}
