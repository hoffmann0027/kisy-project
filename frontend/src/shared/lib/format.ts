// Time and text formatting helpers, in the language on screen.
import { currentLang, intlLocale, t } from "@shared/i18n";

/**
 * How the hour is written: "09:05" on a 24-hour clock, "9:05 AM" — not
 * "09:05 AM" — on a 12-hour one.
 */
export function hourStyle(): "2-digit" | "numeric" {
  const cycle = new Intl.DateTimeFormat(intlLocale(), { hour: "numeric" }).resolvedOptions().hourCycle;
  return cycle === "h11" || cycle === "h12" ? "numeric" : "2-digit";
}

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString(intlLocale(), { hour: hourStyle(), minute: "2-digit" });
}

export function formatDay(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const diffDays = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (diffDays === 0) return t("common.today");
  if (diffDays === 1) return t("common.yesterday");
  return d.toLocaleDateString(intlLocale(), { day: "numeric", month: "long" });
}

export function formatRelative(iso: string): string {
  const d = new Date(iso);
  const now = Date.now();
  const diffSec = Math.round((now - d.getTime()) / 1000);
  if (diffSec < 60) return t("common.justNow");
  if (diffSec < 3600) return t("common.minutesAgo", { count: Math.floor(diffSec / 60) });
  if (diffSec < 86_400) return t("common.hoursAgo", { count: Math.floor(diffSec / 3600) });
  return d.toLocaleDateString(intlLocale(), { day: "numeric", month: "short" });
}

export function initials(name: string): string {
  const parts = name.trim().split(/[\s_]+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

// Deterministic pastel-on-dark color from a string (for avatars).
export function colorFromString(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) hash = str.charCodeAt(i) + ((hash << 5) - hash);
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue} 62% 48%)`;
}

/**
 * Text that continues a sentence ("New password — at least 12 characters"):
 * its first letter lowered, by the rules of the language on screen. Not the
 * whole text — "KISY" stays "KISY" — and not at all in German, where the
 * nouns that start such phrases keep their capital.
 */
export function midSentence(text: string): string {
  if (!text || currentLang() === "de") return text;
  return text.charAt(0).toLocaleLowerCase(intlLocale()) + text.slice(1);
}
