// Time and text formatting helpers, in the language on screen.
import { intlLocale, t } from "@shared/i18n";

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString(intlLocale(), { hour: "2-digit", minute: "2-digit" });
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
