import { afterEach, describe, expect, it } from "vitest";
import { applyDictionary, loadDictionary, t } from "./i18n";
import { detectLang, LANGS } from "./langs";
import { ru, type Key } from "./locales/ru";
import type { Dict, Msg, PluralMsg } from "./types";

afterEach(() => applyDictionary("ru", ru));

describe("choosing the language", () => {
  it("keeps the profile's choice over the device", () => {
    expect(detectLang("en", ["ru-RU"])).toBe("en");
  });

  it("takes the first device language it speaks", () => {
    expect(detectLang(null, ["ja-JP", "ru-RU", "en-US"])).toBe("ru");
    expect(detectLang(null, ["en_GB"])).toBe("en");
  });

  it("falls back to English for a language it does not speak", () => {
    expect(detectLang(null, ["ja-JP"])).toBe("en");
    expect(detectLang("klingon", [])).toBe("en");
  });
});

describe("t", () => {
  it("fills placeholders", () => {
    applyDictionary("ru", { "x.hello": "Привет, {name}!" });
    expect(t("x.hello" as Key, { name: "Анна" })).toBe("Привет, Анна!");
  });

  it("picks the plural form by the language's rules", () => {
    const msgs = { one: "{count} сообщение", few: "{count} сообщения", many: "{count} сообщений", other: "{count} сообщения" };
    applyDictionary("ru", { "x.msgs": msgs });
    expect([1, 3, 5, 21].map((count) => t("x.msgs" as Key, { count }))).toEqual([
      "1 сообщение",
      "3 сообщения",
      "5 сообщений",
      "21 сообщение",
    ]);
    applyDictionary("en", { "x.msgs": { one: "{count} message", other: "{count} messages" } });
    expect([1, 5].map((count) => t("x.msgs" as Key, { count }))).toEqual(["1 message", "5 messages"]);
  });

  it("falls back to Russian for a key the dictionary lacks", () => {
    applyDictionary("en", {});
    expect(t("common.today")).toBe("Сегодня");
  });
});

// Every dictionary on disk, registered in LANGS or not yet: a translator can
// check their work before the language is switched on.
const onDisk = import.meta.glob<{ messages: Dict }>("./locales/*/index.ts", { eager: true });
const dictionaries = Object.entries(onDisk)
  .map(([path, mod]) => [path.split("/")[2], mod.messages] as const)
  .filter(([lang]) => lang !== "ru");

// A translator who drops "{count}" leaves a sentence with no number in it; one
// who writes a plain string where Russian counts loses the plural forms; one
// who gives Polish only "one" and "other" gets "5 wiadomość".
const placeholders = (m: Msg) =>
  [...new Set((typeof m === "string" ? m : Object.values(m).join(" ")).match(/\{\w+\}/g) ?? [])].sort();

describe.each(dictionaries)("the %s dictionary", (lang, dict) => {
  it("has every Russian key, with the same placeholders and plurals", () => {
    for (const [key, source] of Object.entries(ru)) {
      const msg = dict[key];
      expect(msg, key).toBeDefined();
      expect(placeholders(msg), key).toEqual(placeholders(source));
      expect(typeof msg, key).toBe(typeof source);
    }
  });

  it("is a language the app knows how to load once registered", () => {
    expect(["ru", "en", "de", "es", "fr", "nl", "pl", "cs", "uk", "tr"]).toContain(lang);
  });
});

describe.each([["ru", ru as Dict] as const, ...dictionaries])("countable phrases in %s", (lang, dict) => {
  it("have each form the language needs", () => {
    const needed = new Intl.PluralRules(lang).resolvedOptions().pluralCategories;
    for (const [key, msg] of Object.entries(dict)) {
      if (typeof msg === "string") continue;
      for (const category of needed) {
        expect((msg as PluralMsg)[category as keyof PluralMsg], `${key}.${category}`).toBeTypeOf("string");
      }
    }
  });
});

describe.each(LANGS.filter((l) => l !== "ru"))("the registered %s dictionary", (lang) => {
  it("loads", async () => {
    expect(Object.keys(await loadDictionary(lang)).length).toBe(Object.keys(ru).length);
  });
});
