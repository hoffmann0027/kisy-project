import { useMemo, useState } from "react";
import { Bar, CartesianGrid, ComposedChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { t, type Key } from "@shared/i18n";
import { formatKopecks } from "@shared/lib/money";
import type { RatingMonth } from "@shared/api/types";
import { bucketMonthly, type Bucket, type Period } from "@entities/rating/model";
import { Card, CardEmpty, CardTabs } from "./Card";
import { axisTicks, formatCompact, periodLabel, periodTitle } from "./format";
import { MONEY, useThemeInk } from "./palette";

const PERIODS: { value: Period; key: Key }[] = [
  { value: "month", key: "work.rating.periodMonth" },
  { value: "quarter", key: "work.rating.periodQuarter" },
  { value: "year", key: "work.rating.periodYear" },
];

/** Income and expense as bars, profit as the line over them, per period. */
export function FinanceChart({ monthly }: { monthly: RatingMonth[] }) {
  const [period, setPeriod] = useState<Period>("month");
  const ink = useThemeInk();
  const data = useMemo(() => bucketMonthly(monthly, period), [monthly, period]);
  const ticks = useMemo(() => axisTicks(data.flatMap((b) => [b.income, b.expense, b.profit])), [data]);

  return (
    <Card
      title={t("work.rating.chartTitle")}
      className="rchart"
      aside={
        <CardTabs
          value={period}
          options={PERIODS.map((p) => ({ value: p.value, label: t(p.key) }))}
          onChange={setPeriod}
          label={t("work.rating.chartTitle")}
        />
      }
    >
      <div className="rchart__legend" aria-hidden="true">
        <span>
          <i style={{ background: MONEY.income }} /> {t("work.rating.income")}
        </span>
        <span>
          <i style={{ background: MONEY.expense }} /> {t("work.rating.expense")}
        </span>
        <span>
          <i style={{ background: ink.accent }} /> {t("work.rating.profit")}
        </span>
      </div>
      {data.length === 0 ? (
        <CardEmpty>{t("work.rating.noProfitData")}</CardEmpty>
      ) : (
        <div className="rchart__plot">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }} barCategoryGap="28%" barGap={3}>
              <CartesianGrid vertical={false} stroke={ink.grid} />
              <XAxis dataKey="label" tickFormatter={(v: string) => periodLabel(v)} stroke={ink.axis} tick={{ fill: ink.axis, fontSize: 11 }} tickLine={false} axisLine={false} />
              <YAxis
                ticks={ticks}
                domain={[ticks[0], ticks[ticks.length - 1]]}
                tickFormatter={(v: number) => formatCompact(v)}
                stroke={ink.axis}
                tick={{ fill: ink.axis, fontSize: 11 }}
                tickLine={false}
                axisLine={false}
                width={56}
              />
              <Tooltip content={<FinanceTip />} cursor={{ fill: ink.grid }} />
              <Bar dataKey="income" fill={MONEY.income} radius={[4, 4, 0, 0]} maxBarSize={28} />
              <Bar dataKey="expense" fill={MONEY.expense} radius={[4, 4, 0, 0]} maxBarSize={28} />
              <Line type="monotone" dataKey="profit" stroke={ink.accent} strokeWidth={2} dot={{ r: 3, fill: ink.accent, strokeWidth: 0 }} activeDot={{ r: 5 }} />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      )}
    </Card>
  );
}

function FinanceTip({ active, payload }: { active?: boolean; payload?: { payload: Bucket }[] }) {
  const b = payload?.[0]?.payload;
  if (!active || !b) return null;
  return (
    <div className="rc-tip">
      <div className="rc-tip__title">{periodTitle(b.label)}</div>
      <div>
        <i style={{ background: MONEY.income }} /> {t("work.rating.income")} <b>{formatKopecks(b.income)}</b>
      </div>
      <div>
        <i style={{ background: MONEY.expense }} /> {t("work.rating.expense")} <b>{formatKopecks(b.expense)}</b>
      </div>
      <div>
        <i className="rc-tip__accent" /> {t("work.rating.profit")} <b>{formatKopecks(b.profit)}</b>
      </div>
    </div>
  );
}
