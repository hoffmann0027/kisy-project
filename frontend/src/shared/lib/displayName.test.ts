import { describe, expect, it } from "vitest";
import { ApiError } from "@shared/api/envelope";
import {
  DISPLAY_NAME_LENGTH,
  DISPLAY_NAME_LETTERS,
  DISPLAY_NAME_TAKEN,
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
    ["", DISPLAY_NAME_LENGTH],
    ["Я", DISPLAY_NAME_LENGTH],
    ["А".repeat(41), DISPLAY_NAME_LENGTH],
    ["hamza_1", DISPLAY_NAME_LETTERS],
    ["Анна-Мария", DISPLAY_NAME_LETTERS],
    ["Иван 2", DISPLAY_NAME_LETTERS],
    ["Anna 😀", DISPLAY_NAME_LETTERS],
    ["Αλέξης", DISPLAY_NAME_LETTERS],
  ])("refuses %s", (name, problem) => {
    expect(displayNameProblem(name)).toBe(problem);
    expect(displayNameSchema.safeParse(name).success).toBe(false);
  });

  it("turns the server's codes into the form's words", () => {
    expect(displayNameErrorMessage(new ApiError("DISPLAY_NAME_TAKEN", "Имя занято", "r", 409))).toBe(DISPLAY_NAME_TAKEN);
    expect(displayNameErrorMessage(new ApiError("VALIDATION_FAILED", "username is already taken", "r", 409))).toBeNull();
  });
});
