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
