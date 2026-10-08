import { describe, expect, it } from "vitest";
import { colorFromString, formatTime, initials } from "./format";

describe("initials", () => {
  it("takes the first letters of two words", () => {
    expect(initials("Alice Smith")).toBe("AS");
  });
  it("uses two letters for a single word", () => {
    expect(initials("carol")).toBe("CA");
  });
  it("splits on underscores (usernames)", () => {
    expect(initials("john_doe")).toBe("JD");
  });
  it("handles empty input", () => {
    expect(initials("")).toBe("?");
  });
});

describe("colorFromString", () => {
  it("is deterministic", () => {
    expect(colorFromString("alice")).toBe(colorFromString("alice"));
  });
  it("differs for different inputs", () => {
    expect(colorFromString("alice")).not.toBe(colorFromString("bob"));
  });
  it("produces an hsl color", () => {
    expect(colorFromString("x")).toMatch(/^hsl\(\d+ \d+% \d+%\)$/);
  });
});

describe("formatTime", () => {
  it("formats an ISO timestamp as HH:MM", () => {
    expect(formatTime("2026-07-04T09:05:00Z")).toMatch(/^\d{2}:\d{2}$/);
  });
});

describe("midSentence", () => {
  it("lowers the first letter only, and not in German", async () => {
    const { applyDictionary, loadDictionary } = await import("@shared/i18n");
    const { ru } = await import("@shared/i18n/locales/ru");
    const { midSentence } = await import("./format");
    expect(midSentence("Минимум 12 символов, KISY")).toBe("минимум 12 символов, KISY");
    applyDictionary("de", await loadDictionary("de"));
    expect(midSentence("Mindestens 12 Zeichen")).toBe("Mindestens 12 Zeichen");
    applyDictionary("ru", ru);
  });
});

describe("hourStyle", () => {
  it("drops the leading zero only on a 12-hour clock", async () => {
    const { applyDictionary, loadDictionary } = await import("@shared/i18n");
    const { ru } = await import("@shared/i18n/locales/ru");
    const { hourStyle } = await import("./format");
    expect(hourStyle()).toBe("2-digit");
    applyDictionary("en", await loadDictionary("en"));
    // jsdom's navigator.language is en-US: a 12-hour clock.
    expect(hourStyle()).toBe("numeric");
    applyDictionary("ru", ru);
  });
});
