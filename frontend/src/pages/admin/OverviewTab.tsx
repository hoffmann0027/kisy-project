import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { adminApi } from "@shared/api/endpoints";
import type { DashboardOverview } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";
import { intlLocale, t, type Key } from "@shared/i18n";
import { GrowthChart } from "./GrowthChart";
import { auditLabel } from "./auditLabels";
import { FeedbackModal } from "@features/feedback/FeedbackModal";

// "Обзор": what is happening in KISY right now, from real data only — no
// revenue (there are no payments), no countries (no IP is stored), no month
// of uptime (nothing outside the process watches it). See
// backend/internal/dashboard.

export type AdminSection = "overview" | "users" | "invites" | "verification" | "communities" | "reports" | "updates" | "deleted" | "audit";

const CHECK_NAMES: Record<string, Key> = {
  database: "admin.overview.check.database",
  redis: "admin.overview.check.redis",
  files: "admin.overview.check.files",
  turn: "admin.overview.check.turn",
  push_android: "admin.overview.check.pushAndroid",
  push_web: "admin.overview.check.pushWeb",
  captcha: "admin.overview.check.captcha",
};

const REASONS: Record<string, Key> = {
  spam: "admin.reportReason.spam",
  abuse: "admin.reportReason.abuse",
  fraud: "admin.reportReason.fraud",
  illegal: "admin.reportReason.illegal",
  other: "admin.reportReason.other",
};
const TARGETS: Record<string, Key> = {
  user: "admin.overview.target.user",
  message: "admin.overview.target.message",
  post: "admin.overview.target.post",
  community: "admin.overview.target.community",
};
const SEVERITY: Record<string, Key> = {
  high: "admin.overview.severity.high",
  medium: "admin.overview.severity.medium",
  low: "admin.overview.severity.low",
};

/** A label from a table of keys, or the raw value the table does not know. */
const labelOf = (table: Record<string, Key>, value: string) => (table[value] ? t(table[value]) : value);

export function formatBytes(n: number): string {
  if (n >= 1 << 30) return t("admin.overview.bytes.gb", { n: (n / (1 << 30)).toFixed(2) });
  if (n >= 1 << 20) return t("admin.overview.bytes.mb", { n: (n / (1 << 20)).toFixed(n >= 100 << 20 ? 0 : 1) });
  return t("admin.overview.bytes.kb", { n: Math.max(1, Math.round(n / 1024)) });
}

export function uptime(startedAt: string, now = Date.now()): string {
  const min = Math.max(0, Math.floor((now - new Date(startedAt).getTime()) / 60_000));
  const d = Math.floor(min / 1440);
  const h = Math.floor((min % 1440) / 60);
  if (d > 0) return t("admin.overview.uptime.days", { d, h });
  if (h > 0) return t("admin.overview.uptime.hours", { h, m: min % 60 });
  return t("admin.overview.uptime.minutes", { m: min });
}

/** How full a limit is, as a level the meter shows in colour and in words. */
export function usageLevel(used: number, limit: number): { pct: number; level: "ok" | "warn" | "critical"; text: string } {
  const pct = limit > 0 ? Math.min(100, (used / limit) * 100) : 0;
  if (pct >= 90) return { pct, level: "critical", text: t("admin.overview.usage.critical") };
  if (pct >= 70) return { pct, level: "warn", text: t("admin.overview.usage.warn") };
  return { pct, level: "ok", text: t("admin.overview.usage.ok") };
}

export function OverviewTab({ onNavigate }: { onNavigate: (s: AdminSection) => void }) {
  const { data, isPending, isError } = useQuery({
    queryKey: ["admin", "dashboard"],
    queryFn: () => adminApi.dashboard(),
    refetchInterval: 60_000,
  });

  if (isPending) {
    return (
      <div className="dash-loading">
        <Spinner size={28} />
      </div>
    );
  }
  if (isError || !data) return <p className="admin-verify__empty">{t("admin.overview.loadFailed")}</p>;

  return (
    <div className="dash">
      <Kpis o={data} />
      <div className="dash-row dash-row--wide">
        <GrowthChart data={data.growth} />
        <SystemCard o={data} />
      </div>
      <div className="dash-row">
        <InboxCard o={data} onNavigate={onNavigate} />
        <LimitsCard o={data} />
      </div>
      <div className="dash-row">
        <ActivityCard o={data} onNavigate={onNavigate} />
        <ReportsCard o={data} onNavigate={onNavigate} />
      </div>
    </div>
  );
}

function Kpis({ o }: { o: DashboardOverview }) {
  const k = o.kpi;
  const n = (v: number) => v.toLocaleString(intlLocale());
  const tiles = [
    {
      icon: <Icon.Users />,
      label: t("admin.overview.kpi.users"),
      value: n(k.usersTotal),
      sub: t("admin.overview.kpi.usersSub", { day: n(k.usersNew24h), week: n(k.usersNew7d) }),
    },
    {
      icon: <Icon.User />,
      label: t("admin.overview.kpi.active24h"),
      value: n(k.active24h),
      sub: k.usersTotal ? t("admin.overview.kpi.activeShare", { pct: Math.round((k.active24h / k.usersTotal) * 100) }) : "",
    },
    {
      icon: <Icon.Chat />,
      label: t("admin.overview.kpi.messages24h"),
      value: n(k.messages24h),
      sub: t("admin.overview.kpi.messagesSub"),
    },
    {
      icon: <Icon.Community />,
      label: t("admin.overview.kpi.communities"),
      value: n(k.communities),
      sub: t("admin.overview.kpi.groups", { count: k.groups, n: n(k.groups) }),
    },
  ];
  return (
    <div className="dash-kpis">
      {tiles.map((tile) => (
        <div key={tile.label} className="dash-kpi">
          <span className="dash-kpi__icon" aria-hidden="true">
            {tile.icon}
          </span>
          <div className="dash-kpi__label">{tile.label}</div>
          <div className="dash-kpi__value">{tile.value}</div>
          {tile.sub && <div className="dash-kpi__sub">{tile.sub}</div>}
        </div>
      ))}
    </div>
  );
}

function SystemCard({ o }: { o: DashboardOverview }) {
  const down = o.system.checks.filter((c) => c.state === "down").length;
  return (
    <section className="dash-card" aria-label={t("admin.overview.system.title")} id="dash-system">
      <header className="dash-card__head">
        <h3 className="dash-card__title">{t("admin.overview.system.title")}</h3>
      </header>
      <p className={"dash-status " + (down ? "dash-status--critical" : "dash-status--ok")}>
        <span className="dash-dot" aria-hidden="true" />
        {down ? t("admin.overview.system.down", { count: down }) : t("admin.overview.system.allUp")}
      </p>
      <ul className="dash-checks">
        {o.system.checks.map((c) => (
          <li key={c.name} className={`dash-check dash-check--${c.state}`}>
            <span className="dash-dot" aria-hidden="true" />
            <span className="dash-check__name">{labelOf(CHECK_NAMES, c.name)}</span>
            <span className="dash-check__state">
              {c.state === "ok"
                ? c.latencyMs != null
                  ? t("admin.overview.check.latency", { ms: c.latencyMs })
                  : (c.detail ?? t("admin.overview.check.on"))
                : c.state === "down"
                  ? t("admin.overview.check.down")
                  : t("admin.overview.check.off")}
            </span>
          </li>
        ))}
      </ul>
      <p className="dash-card__foot">
        {t("admin.overview.system.footer", {
          version: o.system.version ? o.system.version.slice(0, 7) : t("admin.overview.system.versionUnknown"),
          uptime: uptime(o.system.startedAt),
        })}
      </p>
    </section>
  );
}

function InboxCard({ o, onNavigate }: { o: DashboardOverview; onNavigate: (s: AdminSection) => void }) {
  const qc = useQueryClient();
  // The feedback inbox opens right here; answered entries leave the count
  // when it closes.
  const [feedback, setFeedback] = useState(false);
  const closeFeedback = () => {
    setFeedback(false);
    void qc.invalidateQueries({ queryKey: ["admin", "dashboard"] });
  };
  const items: { label: string; value: number; go?: () => void; hint?: string }[] = [
    { label: t("admin.overview.inbox.openReports"), value: o.inbox.openReports, go: () => onNavigate("reports") },
    { label: t("admin.overview.inbox.unansweredFeedback"), value: o.inbox.unansweredFeedback, go: () => setFeedback(true) },
    { label: t("admin.overview.inbox.joinRequests"), value: o.inbox.pendingJoinRequests, hint: t("admin.overview.inbox.joinRequestsHint") },
  ];
  return (
    <section className="dash-card" aria-label={t("admin.overview.inbox.title")}>
      <header className="dash-card__head">
        <h3 className="dash-card__title">{t("admin.overview.inbox.title")}</h3>
      </header>
      <ul className="dash-inbox">
        {items.map((i) => (
          <li key={i.label}>
            {i.go ? (
              <button className="dash-inbox__row" onClick={i.go}>
                <span>{i.label}</span>
                <strong className={i.value ? "is-due" : ""}>{i.value}</strong>
              </button>
            ) : (
              <div className="dash-inbox__row">
                <span>
                  {i.label}
                  <small>{i.hint}</small>
                </span>
                <strong className={i.value ? "is-due" : ""}>{i.value}</strong>
              </div>
            )}
          </li>
        ))}
      </ul>
      <FeedbackModal open={feedback} onClose={closeFeedback} />
    </section>
  );
}

function Meter({ label, used, limit, note }: { label: string; used: number; limit: number; note?: string }) {
  const u = usageLevel(used, limit);
  return (
    <div className={`dash-meter dash-meter--${u.level}`}>
      <div className="dash-meter__top">
        <span>{label}</span>
        <span>
          {limit > 0 ? t("admin.overview.meter.usedOf", { used: formatBytes(used), limit: formatBytes(limit) }) : formatBytes(used)}
        </span>
      </div>
      {limit > 0 && (
        <div className="dash-meter__bar" role="meter" aria-valuenow={Math.round(u.pct)} aria-valuemin={0} aria-valuemax={100} aria-label={label}>
          <span style={{ width: `${u.pct}%` }} />
        </div>
      )}
      <div className="dash-meter__note">{limit > 0 ? `${Math.round(u.pct)}% · ${u.text}` : note}</div>
    </div>
  );
}

function LimitsCard({ o }: { o: DashboardOverview }) {
  const l = o.limits;
  return (
    <section className="dash-card" aria-label={t("admin.overview.limits.title")}>
      <header className="dash-card__head">
        <h3 className="dash-card__title">{t("admin.overview.limits.title")}</h3>
      </header>
      <Meter label={t("admin.overview.limits.database")} used={l.database.usedBytes} limit={l.database.limitBytes} />
      <div className="dash-meter__note dash-meter__note--block">
        {t("admin.overview.limits.filesInDb", { size: formatBytes(l.filesInDatabase) })}
      </div>
      {l.redis ? (
        <Meter label="Redis (Upstash)" used={l.redis.usedBytes} limit={l.redis.limitBytes} />
      ) : (
        <div className="dash-meter__note">{t("admin.overview.limits.redisUnknown")}</div>
      )}
    </section>
  );
}

function ActivityCard({ o, onNavigate }: { o: DashboardOverview; onNavigate: (s: AdminSection) => void }) {
  return (
    <section className="dash-card" aria-label={t("admin.overview.activity.title")}>
      <header className="dash-card__head">
        <h3 className="dash-card__title">{t("admin.overview.activity.title")}</h3>
        <button className="dash-link" onClick={() => onNavigate("audit")}>
          {t("admin.overview.activity.all")}
        </button>
      </header>
      {o.activity.length === 0 && <p className="admin-verify__empty">{t("admin.overview.activity.empty")}</p>}
      <ul className="dash-list">
        {o.activity.map((a, i) => (
          <li key={i} className="dash-list__row">
            <div>
              <div className="dash-list__title">{auditLabel(a.action)}</div>
              <div className="dash-list__sub">{a.actorName ?? t("admin.overview.activity.system")}</div>
            </div>
            <span className="dash-list__time">{formatRelative(a.createdAt)}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function ReportsCard({ o, onNavigate }: { o: DashboardOverview; onNavigate: (s: AdminSection) => void }) {
  return (
    <section className="dash-card" aria-label={t("admin.overview.reports.title")}>
      <header className="dash-card__head">
        <h3 className="dash-card__title">{t("admin.overview.reports.title")}</h3>
        <button className="dash-link" onClick={() => onNavigate("reports")}>
          {t("admin.overview.reports.all")}
        </button>
      </header>
      {o.reports.length === 0 && <p className="admin-verify__empty">{t("admin.overview.reports.empty")}</p>}
      <ul className="dash-list">
        {o.reports.map((r) => (
          <li key={r.id} className="dash-list__row">
            <div>
              <div className="dash-list__title">{labelOf(REASONS, r.reason)}</div>
              <div className="dash-list__sub">{labelOf(TARGETS, r.targetKind)}</div>
            </div>
            <span className={`dash-sev dash-sev--${r.severity}`}>{SEVERITY[r.severity] ? t(SEVERITY[r.severity]) : undefined}</span>
            <span className="dash-list__time">{formatRelative(r.createdAt)}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}
