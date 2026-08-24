package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrDeviceUnregistered means Firebase rejected the registration token for
// good: the app was uninstalled, its data cleared, or the token rotated. The
// caller must delete the row instead of retrying.
var ErrDeviceUnregistered = errors.New("push: device token is no longer registered")

// fcmScope is the only OAuth scope the sender needs.
const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// serviceAccount is the subset of a Firebase service-account key file that
// matters for minting access tokens.
type serviceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// FCM sends notifications through the Firebase Cloud Messaging HTTP v1 API.
//
// Authentication is a self-signed JWT exchanged for a short-lived OAuth access
// token (the two-legged service-account flow). Doing that here rather than
// pulling in the Google SDK keeps the dependency surface at the JWT library
// the project already signs its own tokens with.
type FCM struct {
	sa     serviceAccount
	client *http.Client

	// sendBase is the FCM endpoint root, overridable in tests.
	sendBase string

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// NewFCM parses a service-account key file. It returns an error when the JSON
// is not a usable service account, so a misconfigured deployment fails at
// startup rather than silently dropping every notification.
func NewFCM(keyJSON []byte) (*FCM, error) {
	var sa serviceAccount
	if err := json.Unmarshal(keyJSON, &sa); err != nil {
		return nil, fmt.Errorf("push: FCM service account is not valid JSON: %w", err)
	}
	if sa.Type != "service_account" {
		return nil, fmt.Errorf("push: FCM credentials must be a service account key, got type %q", sa.Type)
	}
	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("push: FCM service account is missing project_id, client_email or private_key")
	}
	if _, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(sa.PrivateKey)); err != nil {
		return nil, fmt.Errorf("push: FCM service account private key is unreadable: %w", err)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}
	return &FCM{
		sa:       sa,
		client:   &http.Client{Timeout: 10 * time.Second},
		sendBase: "https://fcm.googleapis.com",
	}, nil
}

// ProjectID identifies the Firebase project the sender is bound to.
func (f *FCM) ProjectID() string { return f.sa.ProjectID }

// accessToken returns a cached OAuth token, minting a new one when the current
// one is missing or about to expire.
func (f *FCM) accessToken(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// A minute of slack: a token that expires mid-flight would fail the send.
	if f.token != "" && time.Now().Before(f.tokenExp.Add(-time.Minute)) {
		return f.token, nil
	}

	assertion, err := f.assertion(time.Now())
	if err != nil {
		return "", err
	}
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.sa.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("push: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("push: token exchange: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("push: token exchange returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.AccessToken == "" {
		return "", errors.New("push: token exchange returned no access token")
	}
	f.token = out.AccessToken
	f.tokenExp = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	return f.token, nil
}

// assertion builds the signed JWT that buys an access token.
func (f *FCM) assertion(now time.Time) (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(f.sa.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("push: parse service account key: %w", err)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   f.sa.ClientEmail,
		"scope": fcmScope,
		"aud":   f.sa.TokenURI,
		"iat":   now.Unix(),
		// Google caps the assertion lifetime at one hour.
		"exp": now.Add(time.Hour).Unix(),
	})
	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("push: sign service account assertion: %w", err)
	}
	return signed, nil
}

// fcmMessage mirrors the HTTP v1 request body.
type fcmMessage struct {
	Message struct {
		Token        string            `json:"token"`
		Notification fcmNotification   `json:"notification"`
		Data         map[string]string `json:"data,omitempty"`
		Android      fcmAndroid        `json:"android"`
	} `json:"message"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type fcmAndroid struct {
	Priority     string             `json:"priority"`
	Notification fcmAndroidNotifOpt `json:"notification"`
}

type fcmAndroidNotifOpt struct {
	// One tag for the whole app: a new notification replaces the previous one
	// instead of stacking, matching the Web Push behaviour.
	Tag string `json:"tag,omitempty"`
	// The app creates this channel at startup with high importance, so a
	// message pops up instead of landing silently in the shade. Without it
	// Android would file the push under its low-importance fallback channel.
	ChannelID string `json:"channel_id,omitempty"`
}

// AndroidChannelID must match the channel the mobile client creates
// (frontend/src/shared/lib/nativePush.ts).
const AndroidChannelID = "kisy_messages"

// Send delivers one notification to one device. A returned
// ErrDeviceUnregistered tells the caller to forget the token.
func (f *FCM) Send(ctx context.Context, deviceToken, title, body, link string) error {
	access, err := f.accessToken(ctx)
	if err != nil {
		return err
	}

	var msg fcmMessage
	msg.Message.Token = deviceToken
	msg.Message.Notification = fcmNotification{Title: title, Body: body}
	if link != "" {
		// The app reads this on tap to open the right chat.
		msg.Message.Data = map[string]string{"url": link}
	}
	msg.Message.Android = fcmAndroid{
		Priority:     "HIGH",
		Notification: fcmAndroidNotifOpt{Tag: "kisy", ChannelID: AndroidChannelID},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("push: encode FCM message: %w", err)
	}
	endpoint := fmt.Sprintf("%s/v1/projects/%s/messages:send", f.sendBase, f.sa.ProjectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("push: build FCM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("push: FCM send: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case isUnregistered(resp.StatusCode, respBody):
		return ErrDeviceUnregistered
	default:
		return fmt.Errorf("push: FCM returned %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
}

// isUnregistered reports whether the failure is permanent for this token.
// FCM answers 404/UNREGISTERED for a token that no longer exists and
// 400/INVALID_ARGUMENT for one that never could — both are unrecoverable.
func isUnregistered(status int, body []byte) bool {
	if status != http.StatusNotFound && status != http.StatusBadRequest {
		return false
	}
	var parsed struct {
		Error struct {
			Status  string `json:"status"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		// A 404 with an unparseable body is still a dead endpoint; a 400 is
		// ambiguous, so keep the token and let the send be retried later.
		return status == http.StatusNotFound
	}
	for _, d := range parsed.Error.Details {
		if d.ErrorCode == "UNREGISTERED" || d.ErrorCode == "INVALID_ARGUMENT" {
			return true
		}
	}
	return parsed.Error.Status == "NOT_FOUND"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
