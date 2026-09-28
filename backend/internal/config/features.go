package config

import (
	"slices"
	"strings"
)

// Features is what a deployment actually got, as opposed to what was meant.
//
// Every one of these is switched on by environment variables and, when they
// are missing, used to switch itself off without a word: web push kept
// writing subscriptions nobody would ever deliver to, calls quietly fell back
// to STUN and never connected through a symmetric NAT, and uploads went into
// Postgres instead of object storage. Each was found from a user report, not
// from a log line — the same class of failure as post_media before migration
// 44 (audit D-03).
//
// So the server says out loud, at start and on /ready, which features are
// live; validateProduction refuses a half-configured pair outright.
type Features struct {
	// WebPush: browser push (VAPID pair).
	WebPush bool
	// FCM: push to the packaged mobile app (Firebase service account).
	FCM bool
	// TURN: relayed WebRTC, without which calls behind a symmetric NAT fail.
	TURN bool
	// BlobS3: attachment bytes in object storage rather than in Postgres.
	BlobS3 bool
	// Turnstile: the bot check on sign-up.
	Turnstile bool
}

// Features reports the effective feature state of this configuration.
func (c *Config) Features() Features {
	return Features{
		WebPush:   c.VAPIDPublicKey != "" && c.VAPIDPrivateKey != "",
		FCM:       len(c.FCMServiceAccount) > 0,
		TURN:      len(c.ICE.TURNURLs) > 0 && c.ICE.TURNSecret != "",
		BlobS3:    c.Blob.Enabled(),
		Turnstile: c.Turnstile.Enabled && c.Turnstile.Secret != "",
	}
}

// Map is the shape /ready reports, so an operator can see the same truth
// from outside the process.
func (f Features) Map() map[string]bool {
	return map[string]bool{
		"webpush":   f.WebPush,
		"fcm":       f.FCM,
		"turn":      f.TURN,
		"blob_s3":   f.BlobS3,
		"turnstile": f.Turnstile,
	}
}

// String renders the one-line summary for the startup log, e.g.
// "webpush=off fcm=on turn=off blob_s3=off turnstile=on".
func (f Features) String() string {
	var b strings.Builder
	for i, name := range []string{"webpush", "fcm", "turn", "blob_s3", "turnstile"} {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(name)
		if f.Map()[name] {
			b.WriteString("=on")
		} else {
			b.WriteString("=off")
		}
	}
	return b.String()
}

// halfConfigured lists feature configurations that cannot work as set: one
// half of a pair present and the other missing. In production each is a
// refusal to start — the alternative is a deploy that looks healthy with a
// dead feature inside it (audit D-03).
func (c *Config) halfConfigured() []string {
	var problems []string

	if (c.VAPIDPublicKey == "") != (c.VAPIDPrivateKey == "") {
		problems = append(problems,
			"VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY must be set together — web push is off, but subscriptions are still being stored")
	}
	if len(c.ICE.TURNURLs) > 0 && c.ICE.TURNSecret == "" {
		problems = append(problems,
			"TURN_URLS is set without TURN_SECRET — the server cannot mint relay credentials, so calls behind a symmetric NAT never connect")
	}
	if c.ICE.TURNSecret != "" && len(c.ICE.TURNURLs) == 0 {
		problems = append(problems, "TURN_SECRET is set without TURN_URLS — no relay is offered to clients")
	}

	// Object storage needs all four to work; any subset silently keeps the
	// bytes in Postgres, which is exactly how one free database fills up
	// (audit A-07).
	blob := map[string]string{
		"BLOB_S3_ENDPOINT":   c.Blob.Endpoint,
		"BLOB_S3_BUCKET":     c.Blob.Bucket,
		"BLOB_S3_ACCESS_KEY": c.Blob.AccessKey,
		"BLOB_S3_SECRET_KEY": c.Blob.SecretKey,
	}
	var missing []string
	set := 0
	for name, v := range blob {
		if v == "" {
			missing = append(missing, name)
			continue
		}
		set++
	}
	if set > 0 && len(missing) > 0 {
		slices.Sort(missing)
		problems = append(problems,
			"object storage is half configured — missing "+strings.Join(missing, ", ")+"; attachments would silently stay in Postgres")
	}

	return problems
}
