import { describe, expect, it } from "vitest";
import { ApiError } from "@shared/api/envelope";
import {
  displayNameLengthText,
  displayNameLettersText,
  displayNameTakenText,
  displayNameErrorMessage,
  displayNameProblem,
  displayNameSchema,
  normalizeDisplayName,
} from "./displayName";

// Mirrors backend/internal/users/displayname_test.go: the two must agree, or
// the form accepts what the server refuses (or the other way round).
describe("the display-name rule", () => {
  it.each(["Анна Смирнова", "Anna Smith", "Jo", "Ёлкин", "Łukasz Żółć", "Олексій Їжак", "А".repeat(40)])(
    "accepts %s",
    (name) => expect(displayNameProblem(name)).toBeNull(),
  );

  it("normalizes spaces the way the server stores the name", () => {
    expect(normalizeDisplayName("  Анна   Смирнова ")).toBe("Анна Смирнова");
    expect(displayNameProblem("  Анна   Смирнова ")).toBeNull();
  });

  it.each([
    ["", displayNameLengthText()],
    ["Я", displayNameLengthText()],
    ["А".repeat(41), displayNameLengthText()],
    ["hamza_1", displayNameLettersText()],
    ["Анна-Мария", displayNameLettersText()],
    ["Иван 2", displayNameLettersText()],
    ["Anna 😀", displayNameLettersText()],
    ["Αλέξης", displayNameLettersText()],
  ])("refuses %s", (name, problem) => {
    expect(displayNameProblem(name)).toBe(problem);
    expect(displayNameSchema.safeParse(name).success).toBe(false);
  });

  it("turns the server's codes into the form's words", () => {
    expect(displayNameErrorMessage(new ApiError("DISPLAY_NAME_TAKEN", "Имя занято", "r", 409))).toBe(displayNameTakenText());
    expect(displayNameErrorMessage(new ApiError("VALIDATION_FAILED", "username is already taken", "r", 409))).toBeNull();
  });
});
