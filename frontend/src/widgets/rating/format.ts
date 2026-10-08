// Money and period labels for the rating dashboard, in the language on screen.
import { intlLocale } from "@shared/i18n";
import { formatKopecks } from "@shared/lib/money";

/** "1 234,56 €" with a leading sign for a change: "+1 234,56 €", "−40 €". */
export function formatSigned(cents: number): string {
  const text = formatKopecks(Math.abs(cents));
  if (cents > 0) return `+${text}`;
  if (cents < 0) return `−${text}`;
  return text;
}

/** A short figure for an axis: "2,5 тыс.", "12K" — the currency is in the tooltip. */
export function formatCompact(cents: number): string {
  return new Intl.NumberFormat(intlLocale(), { notation: "compact", maximumFractionDigits: 1 }).format(cents / 100);
}

/**
 * Round ticks for an axis that holds every value, zero included — recharts'
 * own would add a tick below zero to even out the count, and show a loss
 * that never happened.
 */
export function axisTicks(values: number[]): number[] {
  const max = Math.max(0, ...values);
  const min = Math.min(0, ...values);
  const span = max - min || 1;
  const raw = span / 4;
  const mag = 10 ** Math.floor(Math.log10(raw));
  // Never finer than a cent: an empty or all-zero series still gets 0 and 1.
  const step = Math.max(1, [1, 2, 2.5, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? 10 * mag);
  const lo = Math.floor(min / step) * step;
  const hi = Math.max(Math.ceil(max / step) * step, lo + step);
  const ticks: number[] = [];
  for (let v = lo; v <= hi + step * 0.0001; v += step) ticks.push(Math.round(v));
  return ticks;
}

/**
 * A period key as the chart's axis shows it: "2026-10" → "окт.", with the
 * year added when it is not the current one; "2026-Q1" → "Q1 2026"; "2026".
 */
export function periodLabel(key: string, now: Date = new Date()): string {
  const quarter = /^(\d{4})-Q([1-4])$/.exec(key);
  if (quarter) return `Q${quarter[2]} ${quarter[1]}`;
  const month = /^(\d{4})-(\d{2})$/.exec(key);
  if (month) {
    const y = Number(month[1]);
    const d = new Date(y, Number(month[2]) - 1, 1);
    const name = d.toLocaleDateString(intlLocale(), { month: "short" });
    return y === now.getFullYear() ? name : `${name} ${String(y).slice(2)}`;
  }
  return key;
}

/** The full name of a period, for a tooltip: "октябрь 2026", "Q1 2026". */
export function periodTitle(key: string): string {
  const month = /^(\d{4})-(\d{2})$/.exec(key);
  if (month) {
    const d = new Date(Number(month[1]), Number(month[2]) - 1, 1);
    return d.toLocaleDateString(intlLocale(), { month: "long", year: "numeric" });
  }
  return periodLabel(key, new Date(0));
}
