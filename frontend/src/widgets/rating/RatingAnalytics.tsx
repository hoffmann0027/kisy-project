import {
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { formatKopecks } from "@shared/lib/money";
import type { RatingAnalytics as Analytics } from "@shared/api/types";

// Gradient-ish palette matching the KISY brand (cyan → violet).
const COLORS = ["#33c6f6", "#4aa8f6", "#5a8bf6", "#7b34f2", "#9b59f0", "#22c3a6", "#f6a833", "#f65a7b"];

interface Props {
  data: Analytics;
}

// RatingAnalytics renders the two required charts from the "done" column's
// profit ledger: a pie of each project's share of total net profit, and a line
// of total monthly profit across all projects.
export function RatingAnalytics({ data }: Props) {
  const pie = data.perProject
    .filter((p) => p.profitKopecks > 0)
    .map((p) => ({ name: p.title, value: p.profitKopecks }));

  const line = data.monthly.map((m) => ({ month: m.month, rub: m.profitKopecks / 100 }));

  return (
    <div className="rating-analytics">
      <div className="rating-card rating-analytics__panel">
        <div className="rating-analytics__title">Доля чистой прибыли по проектам</div>
        {pie.length === 0 ? (
          <div className="rating-analytics__empty">Пока нет данных о прибыли</div>
        ) : (
          <ResponsiveContainer width="100%" height={240}>
            <PieChart>
              <Pie data={pie} dataKey="value" nameKey="name" innerRadius={50} outerRadius={90} paddingAngle={2}>
                {pie.map((_, i) => (
                  <Cell key={i} fill={COLORS[i % COLORS.length]} stroke="none" />
                ))}
              </Pie>
              <Tooltip
                formatter={(v: number, n) => [formatKopecks(v), n as string]}
                contentStyle={tooltipStyle}
              />
            </PieChart>
          </ResponsiveContainer>
        )}
        {pie.length > 0 && (
          <div className="rating-legend">
            {pie.map((p, i) => (
              <div key={p.name} className="rating-legend__item">
                <span className="rating-legend__dot" style={{ background: COLORS[i % COLORS.length] }} />
                <span className="rating-legend__name">{p.name}</span>
                <span className="rating-legend__val">{formatKopecks(p.value)}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="rating-card rating-analytics__panel">
        <div className="rating-analytics__title">Прибыль по месяцам (все проекты)</div>
        {line.length === 0 ? (
          <div className="rating-analytics__empty">Пока нет данных о прибыли</div>
        ) : (
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={line} margin={{ top: 10, right: 16, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.08)" />
              <XAxis dataKey="month" stroke="#8a8a99" fontSize={12} />
              <YAxis stroke="#8a8a99" fontSize={12} width={64} tickFormatter={(v) => `${v}`} />
              <Tooltip
                formatter={(v: number) => [formatKopecks(Math.round(v * 100)), "Прибыль"]}
                contentStyle={tooltipStyle}
              />
              <Line type="monotone" dataKey="rub" stroke="#5a8bf6" strokeWidth={2.5} dot={{ r: 3 }} />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}

const tooltipStyle = {
  background: "#14141a",
  border: "1px solid rgba(255,255,255,0.1)",
  borderRadius: 10,
  color: "#fff",
} as const;
