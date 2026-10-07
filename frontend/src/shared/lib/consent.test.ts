import { beforeEach, describe, expect, it } from "vitest";
import { clearLocalConsent, hasLocalConsent, saveLocalConsent } from "./consent";
import { PRIVACY_VERSION, RULES_VERSION } from "@shared/config/legalVersions";

describe("consent ticked on this device", () => {
  beforeEach(() => localStorage.clear());

  it("is absent on a fresh install", () => {
    expect(hasLocalConsent()).toBe(false);
  });

  it("is remembered once both boxes are ticked", () => {
    saveLocalConsent();
    expect(hasLocalConsent()).toBe(true);
  });

  it("does not carry over to a newer text", () => {
    // Ticked on last month's rules: this month's have not been seen.
    localStorage.setItem("kisy.consent", JSON.stringify({ privacyVersion: PRIVACY_VERSION, rulesVersion: "2020-01-01" }));
    expect(hasLocalConsent()).toBe(false);
    expect(RULES_VERSION).not.toBe("2020-01-01");
  });

  it("is forgotten at sign-out, so the next person ticks for themselves", () => {
    saveLocalConsent();
    clearLocalConsent();
    expect(hasLocalConsent()).toBe(false);
  });

  it("survives a corrupted store as 'not ticked', not as a crash", () => {
    localStorage.setItem("kisy.consent", "{not json");
    expect(hasLocalConsent()).toBe(false);
  });
});
