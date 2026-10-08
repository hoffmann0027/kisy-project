import type { ReactNode } from "react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

export interface Slice {
  key: string;
  name: string;
  value: number;
  color: string;
}

interface Props {
  slices: Slice[];
  /** What stands in the hole: a total, a count. */
  center: ReactNode;
  /** One row per slice beneath the ring, with whatever figure fits it. */
  rows: { key: string; name: string; color: string; figure: string }[];
  formatValue: (value: number) => string;
}

/** A ring with a figure in the middle and a legend of its parts. */
export function Donut({ slices, center, rows, formatValue }: Props) {
  const drawn = slices.filter((s) => s.value > 0);
  return (
    <div className="rdonut">
      <div className="rdonut__plot">
        {drawn.length > 0 && (
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie data={drawn} dataKey="value" nameKey="name" innerRadius="68%" outerRadius="100%" paddingAngle={drawn.length > 1 ? 3 : 0} stroke="none" isAnimationActive={false}>
                {drawn.map((s) => (
                  <Cell key={s.key} fill={s.color} />
                ))}
              </Pie>
              <Tooltip content={<DonutTip formatValue={formatValue} />} />
            </PieChart>
          </ResponsiveContainer>
        )}
        <div className={"rdonut__center" + (drawn.length === 0 ? " rdonut__center--empty" : "")}>{center}</div>
      </div>
      <ul className="rlegend">
        {rows.map((r) => (
          <li key={r.key} className="rlegend__row">
            <i className="rlegend__dot" style={{ background: r.color }} />
            <span className="rlegend__name">{r.name}</span>
            <span className="rlegend__figure">{r.figure}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function DonutTip({ active, payload, formatValue }: { active?: boolean; payload?: { payload: Slice }[]; formatValue: (v: number) => string }) {
  const s = payload?.[0]?.payload;
  if (!active || !s) return null;
  return (
    <div className="rc-tip">
      <div>
        <i style={{ background: s.color }} /> {s.name} <b>{formatValue(s.value)}</b>
      </div>
    </div>
  );
}
