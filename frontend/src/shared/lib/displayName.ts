import { z } from "zod";
import { ApiError } from "@shared/api/envelope";

// The display-name rule, client side. The server is the authority
// (backend/internal/users/displayname.go, migration 46) — this exists so the
// form says what is wrong before the round trip, in the same words.
//
// Letters of the Latin and Cyrillic alphabets (the same code-point ranges the
// server uses), single spaces between words, 2–40 characters. Unique ignoring
// case and repeated spaces — which only the server can check.

const LETTER = "A-Za-zÀ-ÖØ-öø-ɏЀ-ҁҊ-ӿ";
const PATTERN = new RegExp(`^[${LETTER}]+( [${LETTER}]+)*$`, "u");

export const DISPLAY_NAME_TAKEN = "Имя занято";
export const DISPLAY_NAME_LETTERS = "Только буквы и одиночные пробелы между словами";
export const DISPLAY_NAME_LENGTH = "Имя: от 2 до 40 символов";

/** Trim, and reduce runs of whitespace between words to one space — as the server stores it. */
export function normalizeDisplayName(raw: string): string {
  return raw.trim().split(/\s+/u).filter(Boolean).join(" ");
}

/** The problem with a name, or null when it passes the rule. */
export function displayNameProblem(raw: string): string | null {
  const name = normalizeDisplayName(raw);
  const length = [...name].length;
  if (length < 2 || length > 40) return DISPLAY_NAME_LENGTH;
  if (!PATTERN.test(name)) return DISPLAY_NAME_LETTERS;
  return null;
}

export const displayNameSchema = z.string().superRefine((value, ctx) => {
  const problem = displayNameProblem(value);
  if (problem) ctx.addIssue({ code: z.ZodIssueCode.custom, message: problem });
});

/** The server's verdict on a name, as the message a form shows; null when the error is about something else. */
export function displayNameErrorMessage(err: unknown): string | null {
  if (!(err instanceof ApiError)) return null;
  if (err.code === "DISPLAY_NAME_TAKEN") return DISPLAY_NAME_TAKEN;
  if (err.code === "DISPLAY_NAME_INVALID") return err.message || DISPLAY_NAME_LETTERS;
  return null;
}
