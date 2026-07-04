// Money is handled in integer kopecks end-to-end; only this formatter turns it
// into a human-readable ruble string for display.

const rub = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  maximumFractionDigits: 2,
});

export function formatKopecks(kopecks: number): string {
  return rub.format(kopecks / 100);
}

// parseRublesToKopecks turns a user-typed ruble amount ("1234,50" or "1234.5")
// into integer kopecks, or null if it is not a valid non-negative number.
export function parseRublesToKopecks(input: string): number | null {
  const normalized = input.trim().replace(/\s/g, "").replace(",", ".");
  if (normalized === "") return 0;
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return null;
  return Math.round(parseFloat(normalized) * 100);
}
