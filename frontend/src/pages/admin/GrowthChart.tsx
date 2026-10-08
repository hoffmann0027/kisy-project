import { useEffect, useMemo, useRef, useState } from "react";
import { intlLocale, t } from "@shared/i18n";

// Accounts over time: one series, so no legend — the title names it. The line
// and area wear the theme's accent; grid and axis text stay in muted ink.
// Hovering (or dragging a finger along it) shows that day's total and how many
// joined.

export interface GrowthPoint {
  day: string;
  registrations: number;
  total: number;
}

const RANGES = [7, 30, 90] as const;
type Range = (typeof RANGES)[number];

const H = 220;
const PAD = { top: 16, right: 12, bottom: 26, left: 44 };

/** Round axis steps: 1, 2, 5 × 10ⁿ. */
export function niceTicks(max: number, count = 4): number[] {
  if (max <= 0) return [0, 1];
  const raw = max / count;
  const mag = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? 10 * mag;
  const ticks: number[] = [];
  for (let v = 0; v <= max + step * 0.0001; v += step) ticks.push(v);
  if (ticks[ticks.length - 1] < max) ticks.push(ticks[ticks.length - 1] + step);
  return ticks;
}

const dayLabel = (iso: string) =>
  new Date(iso + "T00:00:00").toLocaleDateString(intlLocale(), { day: "numeric", month: "short" });

export function GrowthChart({ data }: { data: GrowthPoint[] }) {
  const [range, setRange] = useState<Range>(30);
  const [hover, setHover] = useState<number | null>(null);
  const boxRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(600);

  useEffect(() => {
    const el = boxRef.current;
    if (!el || typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(([e]) => setWidth(Math.max(240, Math.round(e.contentRect.width))));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const points = useMemo(() => data.slice(-range), [data, range]);
  const ticks = useMemo(() => niceTicks(Math.max(...points.map((p) => p.total), 1)), [points]);
  const top = ticks[ticks.length - 1];
  const innerW = width - PAD.left - PAD.right;
  const innerH = H - PAD.top - PAD.bottom;
  const x = (i: number) => PAD.left + (points.length <= 1 ? innerW / 2 : (i / (points.length - 1)) * innerW);
  const y = (v: number) => PAD.top + innerH - (v / top) * innerH;

  const line = points.map((p, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(p.total).toFixed(1)}`).join("");
  const area = points.length
    ? `${line}L${x(points.length - 1).toFixed(1)},${y(0)}L${x(0).toFixed(1)},${y(0)}Z`
    : "";
  const labelEvery = Math.max(1, Math.ceil(points.length / 6));

  const onMove = (clientX: number) => {
    const r = boxRef.current?.getBoundingClientRect();
    if (!r || points.length === 0) return;
    const rel = clientX - r.left - PAD.left;
    const i = Math.round((rel / innerW) * (points.length - 1));
    setHover(Math.min(points.length - 1, Math.max(0, i)));
  };

  const h = hover != null ? points[hover] : null;
  const last = points[points.length - 1];

  return (
    <section className="dash-card dash-growth" aria-label={t("admin.growth.title")}>
      <header className="dash-card__head">
        <h3 className="dash-card__title">{t("admin.growth.title")}</h3>
        <div className="dash-range" role="group" aria-label={t("admin.growth.range")}>
          {RANGES.map((r) => (
            <button key={r} className={r === range ? "is-active" : ""} aria-pressed={r === range} onClick={() => setRange(r)}>
              {t("admin.growth.days", { n: r })}
            </button>
          ))}
        </div>
      </header>
      <div
        ref={boxRef}
        className="dash-growth__plot"
        onPointerMove={(e) => onMove(e.clientX)}
        onPointerLeave={() => setHover(null)}
      >
        <svg width={width} height={H} role="img" aria-label={t("admin.growth.ariaTotal", { total: last?.total ?? 0 })}>
          <defs>
            <linearGradient id="dash-growth-fill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" style={{ stopColor: "var(--acc)", stopOpacity: 0.35 }} />
              <stop offset="100%" style={{ stopColor: "var(--acc)", stopOpacity: 0 }} />
            </linearGradient>
          </defs>
          {ticks.map((tick) => (
            <g key={tick}>
              <line className="dash-growth__grid" x1={PAD.left} x2={width - PAD.right} y1={y(tick)} y2={y(tick)} />
              <text className="dash-growth__axis" x={PAD.left - 8} y={y(tick) + 4} textAnchor="end">
                {tick.toLocaleString(intlLocale())}
              </text>
            </g>
          ))}
          {points.map((p, i) =>
            i % labelEvery === 0 || i === points.length - 1 ? (
              <text key={p.day} className="dash-growth__axis" x={x(i)} y={H - 6} textAnchor="middle">
                {dayLabel(p.day)}
              </text>
            ) : null,
          )}
          <path d={area} fill="url(#dash-growth-fill)" />
          <path d={line} className="dash-growth__line" />
          {h && hover != null && (
            <>
              <line className="dash-growth__cross" x1={x(hover)} x2={x(hover)} y1={PAD.top} y2={y(0)} />
              <circle className="dash-growth__dot" cx={x(hover)} cy={y(h.total)} r={5} />
            </>
          )}
        </svg>
        {h && hover != null && (
          <div
            className="dash-tip"
            style={{ left: Math.min(width - 150, Math.max(0, x(hover) - 70)), top: Math.max(0, y(h.total) - 70) }}
          >
            <strong>{h.total.toLocaleString(intlLocale())}</strong> {t("admin.growth.tipUsers", { count: h.total })}
            <div>
              {dayLabel(h.day)} · +{h.registrations}
            </div>
          </div>
        )}
      </div>
    </section>
  );
}
