// Translation at run time: one language at a time, chosen before the app
// renders (main.tsx awaits initI18n) and changed only by reloading, so every
// string on screen — including ones computed once — is in the same language.
import { ru, type Key } from "./locales/ru";
import { detectLang, LANG_STORAGE_KEY, type Lang } from "./langs";
import type { Dict, Msg, PluralMsg } from "./types";

type Params = Record<string, string | number>;

let lang: Lang = "ru";
let dict: Dict = ru;

const loaders: Record<Exclude<Lang, "ru">, () => Promise<{ messages: Dict }>> = {
  en: () => import("./locales/en"),
  de: () => import("./locales/de"),
  es: () => import("./locales/es"),
  fr: () => import("./locales/fr"),
  nl: () => import("./locales/nl"),
  pl: () => import("./locales/pl"),
  cs: () => import("./locales/cs"),
  uk: () => import("./locales/uk"),
  tr: () => import("./locales/tr"),
};

/** The language on screen now. */
export function currentLang(): Lang {
  return lang;
}

/**
 * The locale for dates and numbers: the device's own when it is in the same
 * language (an American phone keeps its 12-hour clock), else the language.
 */
export function intlLocale(): string {
  const device = typeof navigator !== "undefined" ? navigator.language : "";
  return device && device.toLowerCase().split("-")[0] === lang ? device : lang;
}

/** Switch to `next` with its dictionary already loaded (tests, initI18n). */
export function applyDictionary(next: Lang, messages: Dict): void {
  lang = next;
  dict = messages;
  if (typeof document !== "undefined") document.documentElement.lang = next;
}

/** Load a language's dictionary; Russian is built in, the rest load on demand. */
export async function loadDictionary(l: Lang): Promise<Dict> {
  return l === "ru" ? ru : (await loaders[l]()).messages;
}

function readSaved(): string | null {
  try {
    return localStorage.getItem(LANG_STORAGE_KEY);
  } catch {
    return null;
  }
}

/**
 * Pick the language (the profile's choice, else the device's) and load its
 * dictionary. A dictionary that fails to load leaves Russian on screen rather
 * than no app at all.
 */
export async function initI18n(): Promise<Lang> {
  const preferred = typeof navigator !== "undefined" ? (navigator.languages ?? [navigator.language]) : [];
  const chosen = detectLang(readSaved(), preferred);
  try {
    applyDictionary(chosen, await loadDictionary(chosen));
  } catch {
    applyDictionary("ru", ru);
  }
  return lang;
}

/** Remember the choice and restart the interface in it. */
export function setLanguage(next: Lang): void {
  try {
    localStorage.setItem(LANG_STORAGE_KEY, next);
  } catch {
    // Storage refused: the choice lasts until the next start.
  }
  window.location.reload();
}

function pick(msg: Msg, count: number | undefined): string {
  if (typeof msg === "string") return msg;
  const category = new Intl.PluralRules(lang).select(count ?? 0) as keyof PluralMsg;
  return msg[category] ?? msg.other;
}

/**
 * The text for `key` in the current language, with `{name}` placeholders
 * filled from `params`. A countable phrase picks its form by `params.count`.
 */
export function t(key: Key, params?: Params): string {
  const msg = dict[key] ?? ru[key];
  const count = typeof params?.count === "number" ? params.count : undefined;
  const text = pick(msg, count);
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (whole, name: string) => (name in params ? String(params[name]) : whole));
}
