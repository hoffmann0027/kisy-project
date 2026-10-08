// The languages KISY speaks, and how the first one is chosen.

export const LANGS = ["ru", "en"] as const;
export type Lang = (typeof LANGS)[number];

/** Each language named in itself: a person looking for theirs reads it. */
export const LANG_NAMES: Record<Lang, string> = {
  ru: "Русский",
  en: "English",
};

/** A device language the app does not speak falls back to this one. */
export const FALLBACK_LANG: Lang = "en";

export const LANG_STORAGE_KEY = "kisy-lang";

export function isLang(v: unknown): v is Lang {
  return typeof v === "string" && (LANGS as readonly string[]).includes(v);
}

/**
 * The language to start in: the one chosen in the profile, else the first of
 * the device's languages the app speaks, else English.
 */
export function detectLang(saved: string | null, preferred: readonly string[]): Lang {
  if (isLang(saved)) return saved;
  for (const tag of preferred) {
    const primary = tag.toLowerCase().split(/[-_]/)[0];
    if (isLang(primary)) return primary;
  }
  return FALLBACK_LANG;
}
