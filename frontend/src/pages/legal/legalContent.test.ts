import { describe, expect, it } from "vitest";
import type { Dict } from "@shared/i18n/types";
import { RUSSIAN_DOCS, type LegalDocs, type LegalSection } from "./legalContent";

// A translation of a legal document must say the same things in the same
// places: a dropped paragraph is a promise the translation no longer makes.
// Checked for every translation on disk, switched on or not yet.

const translations = Object.entries(import.meta.glob<{ docs: LegalDocs }>("./content/*.ts", { eager: true }))
  .map(([path, mod]) => [path.slice("./content/".length, -".ts".length), mod.docs] as const)
  .filter(([lang]) => lang !== "ru");

const dictionaries = import.meta.glob<{ messages: Dict }>("../../shared/i18n/locales/*/index.ts", { eager: true });

const shape = (sections: LegalSection[]) =>
  sections.map((s) => s.body.map((p) => (p.startsWith("— ") ? "item" : /^\d\./.test(p) ? p.slice(0, 2) : "text")));

const text = (docs: LegalDocs) => JSON.stringify([docs.privacy, docs.deletion, docs.rules]);

describe("the Russian original", () => {
  it("names the word that deletes an account", () => {
    expect(text(RUSSIAN_DOCS)).toContain("УДАЛИТЬ");
  });
});

describe.each(translations)("the %s translation", (lang, docs) => {
  it("has the original's sections, paragraphs and lists, in order", () => {
    expect(shape(docs.privacy)).toEqual(shape(RUSSIAN_DOCS.privacy));
    expect(shape(docs.deletion)).toEqual(shape(RUSSIAN_DOCS.deletion));
    expect(shape(docs.rules)).toEqual(shape(RUSSIAN_DOCS.rules));
  });

  it("keeps the dates, the contact address and the age limits", () => {
    expect(docs.privacyUpdated).toBe(RUSSIAN_DOCS.privacyUpdated);
    expect(docs.rulesUpdated).toBe(RUSSIAN_DOCS.rulesUpdated);
    const all = text(docs);
    expect(all).toContain("kisyandco@gmail.com");
    for (const n of ["13", "16", "30"]) expect(all, n).toContain(n);
    // No Russian left behind (Ukrainian is written in Cyrillic itself).
    if (lang !== "uk") expect(all).not.toMatch(/[А-Яа-яЁё]{3,}/);
  });

  it("tells you to type the same word the app asks for", () => {
    const dict = dictionaries[`../../shared/i18n/locales/${lang}/index.ts`]?.messages;
    expect(dict, `no ${lang} dictionary`).toBeDefined();
    expect(text(docs)).toContain(dict!["account.delete.confirmWord"] as string);
  });
});
