// The legal documents in every language. Russian is the binding text, built
// in; a translation loads with its language and is shown under a note saying
// it is one, with the original a tap away.
import { useEffect, useState } from "react";
import { currentLang, intlLocale, type Lang } from "@shared/i18n";
import { docs as ru } from "./content/ru";

export interface LegalSection {
  title: string;
  /** Paragraphs; one starting with "— " or "1." is drawn as a list item. */
  body: string[];
}

export interface LegalDocs {
  /** ISO dates of the last change, shown in the reader's format. */
  privacyUpdated: string;
  rulesUpdated: string;
  privacy: LegalSection[];
  deletion: LegalSection[];
  rules: LegalSection[];
}

export const RUSSIAN_DOCS = ru;

// Every language the app speaks has its translation: a new language without
// one does not compile.
const translations: Record<Exclude<Lang, "ru">, () => Promise<{ docs: LegalDocs }>> = {
  en: () => import("./content/en"),
  de: () => import("./content/de"),
  es: () => import("./content/es"),
  fr: () => import("./content/fr"),
  nl: () => import("./content/nl"),
  pl: () => import("./content/pl"),
  cs: () => import("./content/cs"),
  uk: () => import("./content/uk"),
  tr: () => import("./content/tr"),
};

/** The day a document changed, in the reader's language. */
export function formatLegalDate(iso: string): string {
  return new Date(`${iso}T12:00:00Z`).toLocaleDateString(intlLocale(), { day: "numeric", month: "long", year: "numeric" });
}

export interface LegalView {
  docs: LegalDocs;
  /** True when docs is a translation, not the binding Russian text. */
  translated: boolean;
}

/**
 * The documents in the language on screen — or the Russian original when
 * `original` is set, or when there is no translation (yet), or while it loads.
 */
export function useLegalDocs(original = false): LegalView {
  const lang = currentLang();
  const load = original || lang === "ru" ? undefined : translations[lang];
  const [loaded, setLoaded] = useState<LegalDocs | null>(null);

  useEffect(() => {
    if (!load) return;
    let live = true;
    load()
      .then((m) => live && setLoaded(m.docs))
      .catch(() => {
        // The translation did not load: the original is still there.
      });
    return () => {
      live = false;
    };
  }, [load]);

  return load && loaded ? { docs: loaded, translated: true } : { docs: ru, translated: false };
}
