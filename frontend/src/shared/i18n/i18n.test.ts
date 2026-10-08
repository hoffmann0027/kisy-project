import { afterEach, describe, expect, it } from "vitest";
import { applyDictionary, loadDictionary, t } from "./i18n";
import { detectLang, LANGS } from "./langs";
import { ru, type Key } from "./locales/ru";
import type { Msg } from "./types";

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

// A translator who drops "{count}" leaves a sentence with no number in it; one
// who writes a plain string where Russian counts loses the plural forms.
const placeholders = (m: Msg) =>
  [...new Set((typeof m === "string" ? m : Object.values(m).join(" ")).match(/\{\w+\}/g) ?? [])].sort();

describe.each(LANGS.filter((l) => l !== "ru"))("the %s dictionary", (lang) => {
  it("has every Russian key, with the same placeholders and plurals", async () => {
    const dict = await loadDictionary(lang);
    for (const [key, source] of Object.entries(ru)) {
      const msg = dict[key];
      expect(msg, key).toBeDefined();
      expect(placeholders(msg), key).toEqual(placeholders(source));
      expect(typeof msg, key).toBe(typeof source);
    }
  });
});
