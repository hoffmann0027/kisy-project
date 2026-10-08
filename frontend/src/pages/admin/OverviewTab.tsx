import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { adminApi } from "@shared/api/endpoints";
import type { DashboardOverview } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";
import { GrowthChart } from "./GrowthChart";
import { auditLabel } from "./auditLabels";
import { FeedbackModal } from "@features/feedback/FeedbackModal";

// "Обзор": what is happening in KISY right now, from real data only — no
// revenue (there are no payments), no countries (no IP is stored), no month
// of uptime (nothing outside the process watches it). See
// backend/internal/dashboard.

export type AdminSection = "overview" | "users" | "invites" | "verification" | "communities" | "reports" | "updates" | "deleted" | "audit";

const CHECK_NAMES: Record<string, string> = {
  database: "База данных",
  redis: "Redis (кеш)",
  files: "Файлы",
  turn: "Звонки (TURN)",
  push_android: "Пуши Android",
  push_web: "Пуши в браузере",
  captcha: "Капча при регистрации",
};

const REASONS: Record<string, string> = {
  spam: "Спам",
  abuse: "Оскорбления",
  fraud: "Мошенничество",
  illegal: "Запрещённый контент",
  other: "Другое",
};
const TARGETS: Record<string, string> = { user: "пользователь", message: "сообщение", post: "запись", community: "сообщество" };
const SEVERITY: Record<string, string> = { high: "Высокая", medium: "Средняя", low: "Низкая" };

export function formatBytes(n: number): string {
  if (n >= 1 << 30) return `${(n / (1 << 30)).toFixed(2)} ГБ`;
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(n >= 100 << 20 ? 0 : 1)} МБ`;
  return `${Math.max(1, Math.round(n / 1024))} КБ`;
}

export function uptime(startedAt: string, now = Date.now()): string {
  const min = Math.max(0, Math.floor((now - new Date(startedAt).getTime()) / 60_000));
  const d = Math.floor(min / 1440);
  const h = Math.floor((min % 1440) / 60);
  if (d > 0) return `${d} д ${h} ч`;
  if (h > 0) return `${h} ч ${min % 60} мин`;
  return `${min} мин`;
}

/** How full a limit is, as a level the meter shows in colour and in words. */
export function usageLevel(used: number, limit: number): { pct: number; level: "ok" | "warn" | "critical"; text: string } {
  const pct = limit > 0 ? Math.min(100, (used / limit) * 100) : 0;
  if (pct >= 90) return { pct, level: "critical", text: "почти заполнено" };
  if (pct >= 70) return { pct, level: "warn", text: "пора думать о месте" };
  return { pct, level: "ok", text: "в норме" };
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
  if (isError || !data) return <p className="admin-verify__empty">Не удалось загрузить обзор</p>;

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
  const n = (v: number) => v.toLocaleString("ru-RU");
  const tiles = [
    { icon: <Icon.Users />, label: "Пользователей", value: n(k.usersTotal), sub: `+${n(k.usersNew24h)} за сутки · +${n(k.usersNew7d)} за неделю` },
    { icon: <Icon.User />, label: "Активны за 24 ч", value: n(k.active24h), sub: k.usersTotal ? `${Math.round((k.active24h / k.usersTotal) * 100)}% от всех` : "" },
    { icon: <Icon.Chat />, label: "Сообщений за 24 ч", value: n(k.messages24h), sub: "только количество — содержимое зашифровано" },
    { icon: <Icon.Community />, label: "Сообществ", value: n(k.communities), sub: `и ${n(k.groups)} групп` },
  ];
  return (
    <div className="dash-kpis">
      {tiles.map((t) => (
        <div key={t.label} className="dash-kpi">
          <span className="dash-kpi__icon" aria-hidden="true">
            {t.icon}
          </span>
          <div className="dash-kpi__label">{t.label}</div>
          <div className="dash-kpi__value">{t.value}</div>
          {t.sub && <div className="dash-kpi__sub">{t.sub}</div>}
        </div>
      ))}
    </div>
  );
}

function SystemCard({ o }: { o: DashboardOverview }) {
  const down = o.system.checks.filter((c) => c.state === "down").length;
  return (
    <section className="dash-card" aria-label="Состояние систем" id="dash-system">
      <header className="dash-card__head">
        <h3 className="dash-card__title">Состояние систем</h3>
      </header>
      <p className={"dash-status " + (down ? "dash-status--critical" : "dash-status--ok")}>
        <span className="dash-dot" aria-hidden="true" />
        {down ? `Не отвечает: ${down}` : "Всё, что включено, работает"}
      </p>
      <ul className="dash-checks">
        {o.system.checks.map((c) => (
          <li key={c.name} className={`dash-check dash-check--${c.state}`}>
            <span className="dash-dot" aria-hidden="true" />
            <span className="dash-check__name">{CHECK_NAMES[c.name] ?? c.name}</span>
            <span className="dash-check__state">
              {c.state === "ok" ? (c.latencyMs != null ? `${c.latencyMs} мс` : (c.detail ?? "включено")) : c.state === "down" ? "не отвечает" : "не настроено"}
            </span>
          </li>
        ))}
      </ul>
      <p className="dash-card__foot">
        Версия {o.system.version ? o.system.version.slice(0, 7) : "неизвестна"} · работает {uptime(o.system.startedAt)}
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
    { label: "Открытые жалобы", value: o.inbox.openReports, go: () => onNavigate("reports") },
    { label: "Отзывы без ответа", value: o.inbox.unansweredFeedback, go: () => setFeedback(true) },
    { label: "Заявки в группы", value: o.inbox.pendingJoinRequests, hint: "решают владельцы групп" },
  ];
  return (
    <section className="dash-card" aria-label="Ждут решения">
      <header className="dash-card__head">
        <h3 className="dash-card__title">Ждут решения</h3>
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
          {formatBytes(used)}
          {limit > 0 && ` из ${formatBytes(limit)}`}
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
    <section className="dash-card" aria-label="Лимиты тарифов">
      <header className="dash-card__head">
        <h3 className="dash-card__title">Лимиты тарифов</h3>
      </header>
      <Meter label="База данных (Neon)" used={l.database.usedBytes} limit={l.database.limitBytes} />
      <div className="dash-meter__note dash-meter__note--block">
        Из них файлы: {formatBytes(l.filesInDatabase)} — пока объектное хранилище не подключено, вложения лежат в базе
      </div>
      {l.redis ? (
        <Meter label="Redis (Upstash)" used={l.redis.usedBytes} limit={l.redis.limitBytes} />
      ) : (
        <div className="dash-meter__note">Redis не сообщает расход памяти</div>
      )}
    </section>
  );
}

function ActivityCard({ o, onNavigate }: { o: DashboardOverview; onNavigate: (s: AdminSection) => void }) {
  return (
    <section className="dash-card" aria-label="Последние события">
      <header className="dash-card__head">
        <h3 className="dash-card__title">Последние события</h3>
        <button className="dash-link" onClick={() => onNavigate("audit")}>
          Весь аудит
        </button>
      </header>
      {o.activity.length === 0 && <p className="admin-verify__empty">Событий пока нет</p>}
      <ul className="dash-list">
        {o.activity.map((a, i) => (
          <li key={i} className="dash-list__row">
            <div>
              <div className="dash-list__title">{auditLabel(a.action)}</div>
              <div className="dash-list__sub">{a.actorName ?? "система"}</div>
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
    <section className="dash-card" aria-label="Открытые жалобы">
      <header className="dash-card__head">
        <h3 className="dash-card__title">Открытые жалобы</h3>
        <button className="dash-link" onClick={() => onNavigate("reports")}>
          Все жалобы
        </button>
      </header>
      {o.reports.length === 0 && <p className="admin-verify__empty">Открытых жалоб нет</p>}
      <ul className="dash-list">
        {o.reports.map((r) => (
          <li key={r.id} className="dash-list__row">
            <div>
              <div className="dash-list__title">{REASONS[r.reason] ?? r.reason}</div>
              <div className="dash-list__sub">{TARGETS[r.targetKind] ?? r.targetKind}</div>
            </div>
            <span className={`dash-sev dash-sev--${r.severity}`}>{SEVERITY[r.severity]}</span>
            <span className="dash-list__time">{formatRelative(r.createdAt)}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}
