import { z } from "zod";
import { ApiError } from "@shared/api/envelope";
import { t } from "@shared/i18n";

// The display-name rule, client side. The server is the authority
// (backend/internal/users/displayname.go, migration 46) — this exists so the
// form says what is wrong before the round trip, in the same words.
//
// Letters of the Latin and Cyrillic alphabets (the same code-point ranges the
// server uses), single spaces between words, 2–40 characters. Unique ignoring
// case and repeated spaces — which only the server can check.

const LETTER = "A-Za-zÀ-ÖØ-öø-ɏЀ-ҁҊ-ӿ";
const PATTERN = new RegExp(`^[${LETTER}]+( [${LETTER}]+)*$`, "u");

/** "The name is taken" — the server's DISPLAY_NAME_TAKEN, in words. */
export function displayNameTakenText(): string {
  return t("common.displayName.taken");
}
/** The rule a name broke when it has something other than letters and single spaces. */
export function displayNameLettersText(): string {
  return t("common.displayName.letters");
}
/** The rule a name broke when it is too short or too long. */
export function displayNameLengthText(): string {
  return t("common.displayName.length");
}

/** Trim, and reduce runs of whitespace between words to one space — as the server stores it. */
export function normalizeDisplayName(raw: string): string {
  return raw.trim().split(/\s+/u).filter(Boolean).join(" ");
}

/** The problem with a name, or null when it passes the rule. */
export function displayNameProblem(raw: string): string | null {
  const name = normalizeDisplayName(raw);
  const length = [...name].length;
  if (length < 2 || length > 40) return displayNameLengthText();
  if (!PATTERN.test(name)) return displayNameLettersText();
  return null;
}

export const displayNameSchema = z.string().superRefine((value, ctx) => {
  const problem = displayNameProblem(value);
  if (problem) ctx.addIssue({ code: z.ZodIssueCode.custom, message: problem });
});

/** The server's verdict on a name, as the message a form shows; null when the error is about something else. */
export function displayNameErrorMessage(err: unknown): string | null {
  if (!(err instanceof ApiError)) return null;
  if (err.code === "DISPLAY_NAME_TAKEN") return displayNameTakenText();
  if (err.code === "DISPLAY_NAME_INVALID") return err.message || displayNameLettersText();
  return null;
}
