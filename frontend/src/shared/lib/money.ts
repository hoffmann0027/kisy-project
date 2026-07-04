// Money is handled in integer cents end-to-end; only this formatter turns it
// into a human-readable euro string for display.

const eur = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "EUR",
  maximumFractionDigits: 2,
});

// formatKopecks formats an integer amount of euro cents as "1 234,56 €".
export function formatKopecks(cents: number): string {
  return eur.format(cents / 100);
}

// parseRublesToKopecks turns a user-typed euro amount ("1234,50" or "1234.5")
// into integer cents, or null if it is not a valid non-negative number.
export function parseRublesToKopecks(input: string): number | null {
  const normalized = input.trim().replace(/\s/g, "").replace(",", ".");
  if (normalized === "") return 0;
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return null;
  return Math.round(parseFloat(normalized) * 100);
}
