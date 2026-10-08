// Money is handled in integer cents end-to-end; only this formatter turns it
// into a human-readable euro string for display.

import { intlLocale } from "@shared/i18n";

// Built on first use, not at load: the language is chosen after this module
// is evaluated. Kept per locale so a formatter is not rebuilt on every call.
let eur: { locale: string; format: Intl.NumberFormat } | null = null;

function euroFormat(): Intl.NumberFormat {
  const locale = intlLocale();
  if (eur?.locale !== locale) {
    eur = {
      locale,
      format: new Intl.NumberFormat(locale, { style: "currency", currency: "EUR", maximumFractionDigits: 2 }),
    };
  }
  return eur.format;
}

// formatKopecks formats an integer amount of euro cents as "1 234,56 €"
// (in Russian; "€1,234.56" in English).
export function formatKopecks(cents: number): string {
  return euroFormat().format(cents / 100);
}

// parseRublesToKopecks turns a user-typed euro amount ("1234,50" or "1234.5")
// into integer cents, or null if it is not a valid non-negative number.
export function parseRublesToKopecks(input: string): number | null {
  const normalized = input.trim().replace(/\s/g, "").replace(",", ".");
  if (normalized === "") return 0;
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return null;
  return Math.round(parseFloat(normalized) * 100);
}
