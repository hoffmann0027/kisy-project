package push

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// newTestAccount builds a syntactically real service-account key whose token
// endpoint points at the given URL.
func newTestAccount(t *testing.T, tokenURI string) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	raw, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "kisy-test",
		"client_email": "pusher@kisy-test.iam.gserviceaccount.com",
		"private_key":  string(pemBytes),
		"token_uri":    tokenURI,
	})
	if err != nil {
		t.Fatalf("marshal account: %v", err)
	}
	return raw, key
}

// fcmServer serves both the OAuth token endpoint and the FCM send endpoint.
type fcmServer struct {
	*httptest.Server
	tokenCalls atomic.Int32
	sendCalls  atomic.Int32
	lastBody   atomic.Value // json.RawMessage
	lastAuth   atomic.Value // string
	sendStatus atomic.Int32
	sendBody   atomic.Value // string
	expiresIn  atomic.Int32
}

func newFCMServer(t *testing.T) *fcmServer {
	t.Helper()
	s := &fcmServer{}
	s.sendStatus.Store(http.StatusOK)
	s.sendBody.Store(`{"name":"projects/kisy-test/messages/1"}`)
	s.expiresIn.Store(3600)
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		s.tokenCalls.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Errorf("grant_type = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"access-%d","expires_in":%d}`, s.tokenCalls.Load(), s.expiresIn.Load())
	})
	mux.HandleFunc("/v1/projects/kisy-test/messages:send", func(w http.ResponseWriter, r *http.Request) {
		s.sendCalls.Add(1)
		s.lastAuth.Store(r.Header.Get("Authorization"))
		var body json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode send body: %v", err)
		}
		s.lastBody.Store(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(s.sendStatus.Load()))
		fmt.Fprint(w, s.sendBody.Load().(string))
	})
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

// newTestFCM wires a sender against the fake Google endpoints.
func newTestFCM(t *testing.T, srv *fcmServer) *FCM {
	t.Helper()
	raw, _ := newTestAccount(t, srv.URL+"/token")
	f, err := NewFCM(raw)
	if err != nil {
		t.Fatalf("NewFCM: %v", err)
	}
	f.sendBase = srv.URL
	return f
}

func TestNewFCMRejectsUnusableCredentials(t *testing.T) {
	cases := map[string]string{
		"not json":       `nope`,
		"wrong type":     `{"type":"authorized_user","project_id":"p","client_email":"e","private_key":"k"}`,
		"missing fields": `{"type":"service_account","project_id":"p"}`,
		"bad key":        `{"type":"service_account","project_id":"p","client_email":"e","private_key":"not a pem"}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewFCM([]byte(raw)); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

func TestFCMSendBuildsSignedRequest(t *testing.T) {
	srv := newFCMServer(t)
	raw, key := newTestAccount(t, srv.URL+"/token")
	f, err := NewFCM(raw)
	if err != nil {
		t.Fatalf("NewFCM: %v", err)
	}
	f.sendBase = srv.URL

	if err := f.Send(context.Background(), "device-1", "Иван", "Привет", "/chats/42"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// The access token from the exchange must authorize the send.
	if got := srv.lastAuth.Load().(string); got != "Bearer access-1" {
		t.Errorf("Authorization = %q", got)
	}

	var sent fcmMessage
	if err := json.Unmarshal(srv.lastBody.Load().(json.RawMessage), &sent); err != nil {
		t.Fatalf("decode sent message: %v", err)
	}
	if sent.Message.Token != "device-1" {
		t.Errorf("token = %q", sent.Message.Token)
	}
	if sent.Message.Notification.Title != "Иван" || sent.Message.Notification.Body != "Привет" {
		t.Errorf("notification = %+v", sent.Message.Notification)
	}
	if sent.Message.Data["url"] != "/chats/42" {
		t.Errorf("data url = %q", sent.Message.Data["url"])
	}
	if sent.Message.Android.Priority != "HIGH" {
		t.Errorf("android priority = %q", sent.Message.Android.Priority)
	}
	// Wrong channel id and the phone files the message silently.
	if sent.Message.Android.Notification.ChannelID != AndroidChannelID {
		t.Errorf("android channel = %q", sent.Message.Android.Notification.ChannelID)
	}

	// The assertion must be verifiable with the account key and carry the
	// claims Google requires.
	assertion, err := f.assertion(time.Now())
	if err != nil {
		t.Fatalf("assertion: %v", err)
	}
	parsed, err := jwt.Parse(assertion, func(*jwt.Token) (any, error) { return &key.PublicKey, nil },
		jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		t.Fatalf("parse assertion: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["iss"] != "pusher@kisy-test.iam.gserviceaccount.com" {
		t.Errorf("iss = %v", claims["iss"])
	}
	if claims["scope"] != fcmScope {
		t.Errorf("scope = %v", claims["scope"])
	}
	if claims["aud"] != srv.URL+"/token" {
		t.Errorf("aud = %v", claims["aud"])
	}
}

func TestFCMReusesAccessTokenUntilExpiry(t *testing.T) {
	srv := newFCMServer(t)
	f := newTestFCM(t, srv)

	for i := 0; i < 3; i++ {
		if err := f.Send(context.Background(), "device-1", "t", "b", ""); err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
	}
	if got := srv.tokenCalls.Load(); got != 1 {
		t.Fatalf("token exchanges = %d, want 1", got)
	}

	// An expired token is re-minted rather than reused.
	f.mu.Lock()
	f.tokenExp = time.Now()
	f.mu.Unlock()
	if err := f.Send(context.Background(), "device-1", "t", "b", ""); err != nil {
		t.Fatalf("Send after expiry: %v", err)
	}
	if got := srv.tokenCalls.Load(); got != 2 {
		t.Fatalf("token exchanges after expiry = %d, want 2", got)
	}
	if got := srv.sendCalls.Load(); got != 4 {
		t.Fatalf("sends = %d, want 4", got)
	}
}

func TestFCMSendReportsUnregisteredDevices(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{
			name:   "uninstalled app",
			status: http.StatusNotFound,
			body:   `{"error":{"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`,
			want:   ErrDeviceUnregistered,
		},
		{
			name:   "malformed token",
			status: http.StatusBadRequest,
			body:   `{"error":{"status":"INVALID_ARGUMENT","details":[{"errorCode":"INVALID_ARGUMENT"}]}}`,
			want:   ErrDeviceUnregistered,
		},
		{
			// A quota or outage must not cost the user their registration.
			name:   "server unavailable",
			status: http.StatusServiceUnavailable,
			body:   `{"error":{"status":"UNAVAILABLE"}}`,
			want:   nil,
		},
		{
			name:   "quota exceeded",
			status: http.StatusTooManyRequests,
			body:   `{"error":{"status":"RESOURCE_EXHAUSTED"}}`,
			want:   nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newFCMServer(t)
			srv.sendStatus.Store(int32(tc.status))
			srv.sendBody.Store(tc.body)
			f := newTestFCM(t, srv)

			err := f.Send(context.Background(), "device-1", "t", "b", "")
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if tc.want == nil && errors.Is(err, ErrDeviceUnregistered) {
				t.Fatalf("transient failure must not drop the token: %v", err)
			}
		})
	}
}

func TestFCMSendSurfacesTokenExchangeFailure(t *testing.T) {
	srv := newFCMServer(t)
	f := newTestFCM(t, srv)
	// Point the exchange at an endpoint that answers 401.
	unauthorized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"invalid_grant"}`)
	}))
	defer unauthorized.Close()
	f.sa.TokenURI = unauthorized.URL

	err := f.Send(context.Background(), "device-1", "t", "b", "")
	if err == nil || !strings.Contains(err.Error(), "token exchange returned 401") {
		t.Fatalf("error = %v, want the token exchange failure", err)
	}
	if got := srv.sendCalls.Load(); got != 0 {
		t.Fatalf("sends = %d, want 0 — no token, no request", got)
	}
}
