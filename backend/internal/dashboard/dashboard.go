// Package dashboard is the "Обзор" of the admin panel: what is happening in
// KISY right now, from real data only.
//
// The owner's reference showed more than KISY can honestly count, and the
// redesign kept only what it can (October 2026):
//
//   - no revenue — there are no payments; no countries — no IP is stored;
//   - no month-long uptime — nothing outside the process watches it, so the
//     panel shows how long this process has been up and what it runs;
//   - service status is a real check made now (a ping, a STUN probe of the
//     relay), or "configured / not configured" where that is all there is;
//   - the free-tier limits are shown with how close each one is, because the
//     database filling up is how this deployment would actually go down.
//
// CEO only: mounted under /admin.
package dashboard

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/i18n"
)

// GrowthDays is how far back the registrations chart reaches.
const GrowthDays = 90

// Features is what the deployment has configured (config.Features).
type Features struct {
	WebPush   bool
	FCM       bool
	TURN      bool
	BlobS3    bool
	Turnstile bool
}

// Settings describe the deployment around the process.
type Settings struct {
	// Version is the deployed commit (Render sets RENDER_GIT_COMMIT).
	Version string
	// StartedAt is when this process started.
	StartedAt time.Time
	Features  Features
	// TURNURLs: the first one is probed with a STUN binding request.
	TURNURLs []string
	// DBLimitBytes / RedisLimitBytes: the plan's ceilings (Neon free 0.5 GB,
	// Upstash free 256 MB by default); 0 means unknown.
	DBLimitBytes    int64
	RedisLimitBytes int64
}

type Service struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	set  Settings
	now  func() time.Time
}

func NewService(pool *pgxpool.Pool, rdb *redis.Client, set Settings) *Service {
	return &Service{pool: pool, rdb: rdb, set: set, now: time.Now}
}

// KPI are the headline numbers.
type KPI struct {
	UsersTotal  int `json:"usersTotal"`
	UsersNew24h int `json:"usersNew24h"`
	UsersNew7d  int `json:"usersNew7d"`
	// Active24h: signed-in accounts that used the app in the last 24 hours
	// (a session refreshed, or a socket closed).
	Active24h   int `json:"active24h"`
	Messages24h int `json:"messages24h"`
	Communities int `json:"communities"`
	Groups      int `json:"groups"`
}

// Day is one point of the registrations chart.
type Day struct {
	Day           string `json:"day"`
	Registrations int    `json:"registrations"`
	// Total is how many accounts existed at the end of the day.
	Total int `json:"total"`
}

// Inbox is what waits for the CEO.
type Inbox struct {
	OpenReports         int `json:"openReports"`
	UnansweredFeedback  int `json:"unansweredFeedback"`
	PendingJoinRequests int `json:"pendingJoinRequests"`
}

// Activity is one recent audit event.
type Activity struct {
	Action     string    `json:"action"`
	ActorName  *string   `json:"actorName"`
	TargetType *string   `json:"targetType"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Report is one open report, newest first.
type Report struct {
	ID         string    `json:"id"`
	TargetKind string    `json:"targetKind"`
	Reason     string    `json:"reason"`
	Severity   string    `json:"severity"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Check is one service's state right now.
type Check struct {
	Name string `json:"name"`
	// State: ok | down | off (not configured).
	State     string `json:"state"`
	LatencyMs *int64 `json:"latencyMs,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// System is what runs and how it is.
type System struct {
	Version   string    `json:"version"`
	StartedAt time.Time `json:"startedAt"`
	Checks    []Check   `json:"checks"`
}

// Usage is one limit and how much of it is used; Limit 0 means unknown.
type Usage struct {
	UsedBytes  int64 `json:"usedBytes"`
	LimitBytes int64 `json:"limitBytes"`
}

// Limits are the free-tier ceilings this deployment lives under.
type Limits struct {
	Database Usage `json:"database"`
	// FilesInDatabase: attachment, avatar and note bytes held in Postgres —
	// all of it while object storage is not configured.
	FilesInDatabase int64  `json:"filesInDatabase"`
	Redis           *Usage `json:"redis"`
}

// Overview is the whole "Обзор" page.
type Overview struct {
	KPI      KPI        `json:"kpi"`
	Growth   []Day      `json:"growth"`
	Inbox    Inbox      `json:"inbox"`
	Activity []Activity `json:"activity"`
	Reports  []Report   `json:"reports"`
	System   System     `json:"system"`
	Limits   Limits     `json:"limits"`
}

// severity: how urgent a report's reason is.
func severity(reason string) string {
	switch reason {
	case "fraud", "illegal":
		return "high"
	case "abuse":
		return "medium"
	default:
		return "low"
	}
}

// Overview gathers the page. A failure in a measurement that can fail on its
// own (Redis INFO, the relay probe) is shown on the page, not returned.
func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	o := &Overview{}
	if err := s.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM users WHERE deleted_at IS NULL),
		  (SELECT count(*) FROM users WHERE deleted_at IS NULL AND created_at > now() - interval '24 hours'),
		  (SELECT count(*) FROM users WHERE deleted_at IS NULL AND created_at > now() - interval '7 days'),
		  (SELECT count(*) FROM (
		      SELECT user_id FROM sessions WHERE last_used_at > now() - interval '24 hours'
		      UNION
		      SELECT id FROM users WHERE last_seen_at > now() - interval '24 hours') a),
		  (SELECT count(*) FROM messages WHERE created_at > now() - interval '24 hours'),
		  (SELECT count(*) FROM groups WHERE kind = 'community' AND NOT is_archived),
		  (SELECT count(*) FROM groups WHERE kind = 'group' AND NOT is_archived),
		  (SELECT count(*) FROM reports WHERE status = 'open'),
		  (SELECT count(*) FROM feedback WHERE replied_at IS NULL),
		  (SELECT count(*) FROM group_join_requests WHERE status = 'pending')`).Scan(
		&o.KPI.UsersTotal, &o.KPI.UsersNew24h, &o.KPI.UsersNew7d, &o.KPI.Active24h, &o.KPI.Messages24h,
		&o.KPI.Communities, &o.KPI.Groups,
		&o.Inbox.OpenReports, &o.Inbox.UnansweredFeedback, &o.Inbox.PendingJoinRequests); err != nil {
		return nil, fmt.Errorf("dashboard: counts: %w", err)
	}

	growth, err := s.growth(ctx)
	if err != nil {
		return nil, err
	}
	o.Growth = growth

	rows, err := s.pool.Query(ctx, `
		SELECT a.action, u.display_name, a.target_type, a.created_at
		FROM audit_logs a LEFT JOIN users u ON u.id = a.actor_id
		ORDER BY a.created_at DESC LIMIT 8`)
	if err != nil {
		return nil, fmt.Errorf("dashboard: activity: %w", err)
	}
	o.Activity, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Activity, error) {
		var a Activity
		return a, r.Scan(&a.Action, &a.ActorName, &a.TargetType, &a.CreatedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("dashboard: activity: %w", err)
	}

	rows, err = s.pool.Query(ctx, `
		SELECT id::text, target_kind, reason, created_at FROM reports
		WHERE status = 'open' ORDER BY created_at DESC LIMIT 5`)
	if err != nil {
		return nil, fmt.Errorf("dashboard: reports: %w", err)
	}
	o.Reports, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Report, error) {
		var rep Report
		err := r.Scan(&rep.ID, &rep.TargetKind, &rep.Reason, &rep.CreatedAt)
		rep.Severity = severity(rep.Reason)
		return rep, err
	})
	if err != nil {
		return nil, fmt.Errorf("dashboard: reports: %w", err)
	}

	if o.Limits, err = s.limits(ctx); err != nil {
		return nil, err
	}
	o.System = s.system(ctx)
	return o, nil
}

// growth is registrations per day for GrowthDays, every day present, with
// the running total of accounts.
func (s *Service) growth(ctx context.Context) ([]Day, error) {
	var before int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM users
		WHERE deleted_at IS NULL AND created_at < date_trunc('day', now()) - make_interval(days => $1 - 1)`,
		GrowthDays).Scan(&before); err != nil {
		return nil, fmt.Errorf("dashboard: growth base: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(d, 'YYYY-MM-DD'), count(u.id)
		FROM generate_series(date_trunc('day', now()) - make_interval(days => $1 - 1), date_trunc('day', now()), interval '1 day') d
		LEFT JOIN users u ON u.deleted_at IS NULL AND u.created_at >= d AND u.created_at < d + interval '1 day'
		GROUP BY d ORDER BY d`, GrowthDays)
	if err != nil {
		return nil, fmt.Errorf("dashboard: growth: %w", err)
	}
	days, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Day, error) {
		var d Day
		return d, r.Scan(&d.Day, &d.Registrations)
	})
	if err != nil {
		return nil, fmt.Errorf("dashboard: growth: %w", err)
	}
	total := before
	for i := range days {
		total += days[i].Registrations
		days[i].Total = total
	}
	return days, nil
}

// fileTables hold uploaded bytes in Postgres when object storage is off.
var fileTables = []string{"attachments", "avatars", "notes"}

func (s *Service) limits(ctx context.Context) (Limits, error) {
	l := Limits{Database: Usage{LimitBytes: s.set.DBLimitBytes}}
	if err := s.pool.QueryRow(ctx, `
		SELECT pg_database_size(current_database()),
		       COALESCE((SELECT sum(pg_total_relation_size(to_regclass(t)))
		                 FROM unnest($1::text[]) t WHERE to_regclass(t) IS NOT NULL), 0)::bigint`,
		fileTables).Scan(&l.Database.UsedBytes, &l.FilesInDatabase); err != nil {
		return l, fmt.Errorf("dashboard: sizes: %w", err)
	}
	if s.rdb != nil {
		if used, err := redisUsedMemory(ctx, s.rdb); err == nil {
			l.Redis = &Usage{UsedBytes: used, LimitBytes: s.set.RedisLimitBytes}
		}
	}
	return l, nil
}

func redisUsedMemory(ctx context.Context, rdb *redis.Client) (int64, error) {
	info, err := rdb.Info(ctx, "memory").Result()
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(info, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "used_memory:"); ok {
			return strconv.ParseInt(v, 10, 64)
		}
	}
	return 0, errors.New("dashboard: used_memory not reported")
}

func ms(d time.Duration) *int64 {
	v := d.Milliseconds()
	return &v
}

func (s *Service) system(ctx context.Context) System {
	sys := System{Version: s.set.Version, StartedAt: s.set.StartedAt}
	lang := i18n.FromContext(ctx)
	check := func(name string, f func(context.Context) error) {
		c, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		start := s.now()
		if err := f(c); err != nil {
			sys.Checks = append(sys.Checks, Check{Name: name, State: "down", Detail: i18n.T(lang, "dashboard.down")})
			return
		}
		sys.Checks = append(sys.Checks, Check{Name: name, State: "ok", LatencyMs: ms(s.now().Sub(start))})
	}
	configured := func(name string, on bool, detail string) {
		state := "ok"
		if !on {
			state = "off"
		}
		sys.Checks = append(sys.Checks, Check{Name: name, State: state, Detail: detail})
	}

	check("database", func(c context.Context) error { return s.pool.Ping(c) })
	if s.rdb != nil {
		check("redis", func(c context.Context) error { return s.rdb.Ping(c).Err() })
	}
	if s.set.Features.BlobS3 {
		configured("files", true, i18n.T(lang, "dashboard.objectStorage"))
	} else {
		configured("files", true, i18n.T(lang, "dashboard.inDatabase"))
	}
	if s.set.Features.TURN && len(s.set.TURNURLs) > 0 {
		check("turn", func(c context.Context) error { return stunProbe(c, s.set.TURNURLs[0]) })
	} else {
		configured("turn", false, "")
	}
	configured("push_android", s.set.Features.FCM, "")
	configured("push_web", s.set.Features.WebPush, "")
	configured("captcha", s.set.Features.Turnstile, "")
	return sys
}

// stunProbe sends a STUN binding request (RFC 5389) to the relay over UDP and
// waits for a success response: the relay is up and reachable from here.
func stunProbe(ctx context.Context, turnURL string) error {
	addr, err := stunAddr(turnURL)
	if err != nil {
		return err
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	req := make([]byte, 20)
	binary.BigEndian.PutUint16(req[0:], 0x0001) // Binding Request
	binary.BigEndian.PutUint32(req[4:], 0x2112A442)
	if _, err := rand.Read(req[8:20]); err != nil {
		return err
	}
	if _, err := conn.Write(req); err != nil {
		return err
	}
	resp := make([]byte, 512)
	n, err := conn.Read(resp)
	if err != nil {
		return err
	}
	if n < 20 || binary.BigEndian.Uint16(resp[0:]) != 0x0101 || string(resp[8:20]) != string(req[8:20]) {
		return errors.New("dashboard: not a STUN binding success")
	}
	return nil
}

// stunAddr turns "turn:host:3478?transport=udp" into "host:3478".
func stunAddr(turnURL string) (string, error) {
	rest, ok := strings.CutPrefix(turnURL, "turn:")
	if !ok {
		if rest, ok = strings.CutPrefix(turnURL, "stun:"); !ok {
			return "", fmt.Errorf("dashboard: not a turn/stun url: %q", turnURL)
		}
	}
	if i := strings.IndexByte(rest, '?'); i >= 0 {
		rest = rest[:i]
	}
	if _, _, err := net.SplitHostPort(rest); err != nil {
		rest = net.JoinHostPort(rest, "3478")
	}
	if _, err := url.Parse("udp://" + rest); err != nil {
		return "", err
	}
	return rest, nil
}
