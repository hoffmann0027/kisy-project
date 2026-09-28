import { describe, expect, it } from "vitest";
import { passwordProblem } from "./password";

/**
 * Audit D-14: the sign-up form and the server disagreed about what a password
 * is. The form required an ASCII letter, so "парольнадежный1" — perfectly
 * acceptable to the API — was refused with "нужна хотя бы одна буква", and the
 * two change-password screens checked the length only.
 *
 * The table below is the same one the server's policy test uses
 * (backend/internal/auth/password/policy_test.go); if the two ever disagree,
 * one of them is lying to the person typing.
 */
describe("passwordProblem", () => {
  it("accepts a Cyrillic password, as the server does", () => {
    expect(passwordProblem("парольнадежный1")).toBeNull();
  });

  it("accepts latin with a digit", () => {
    expect(passwordProblem("korrekt-parol1")).toBeNull();
  });

  it("accepts exactly the minimum length", () => {
    expect(passwordProblem("abcdefghijk1")).toBeNull();
  });

  it("refuses one character short", () => {
    expect(passwordProblem("abcdefghij1")).toBe("Минимум 12 символов");
  });

  it("refuses a password without a digit", () => {
    expect(passwordProblem("парольбезцифры")).toBe("Нужна хотя бы одна цифра");
  });

  it("refuses a password without a letter", () => {
    expect(passwordProblem("123456789012")).toBe("Нужна хотя бы одна буква");
  });

  it("counts characters, not bytes: 128 Cyrillic characters are allowed", () => {
    expect(passwordProblem("я".repeat(127) + "1")).toBeNull();
  });

  it("counts characters, not code units: an emoji is one character", () => {
    // Six emoji plus "a1" is 8 characters — too short — but 14 UTF-16 code
    // units, so a check on .length would have accepted it.
    expect(passwordProblem("😀".repeat(6) + "a1")).toBe("Минимум 12 символов");
  });

  it("does not refuse a long password just because emoji take two code units", () => {
    // 67 characters, 132 code units: a check on .length called it too long.
    expect(passwordProblem("😀".repeat(65) + "a1")).toBeNull();
  });

  it("refuses one character over the limit", () => {
    expect(passwordProblem("a".repeat(128) + "1")).toBe("Не более 128 символов");
  });
});
