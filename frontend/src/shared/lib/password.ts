/**
 * The password rule, once, mirroring the server (backend
 * internal/auth/password/policy.go).
 *
 * There used to be four different rules on this side: the sign-up form
 * demanded an ASCII letter, so a Cyrillic password was refused although the
 * server accepts it; the change-password screens checked the length only; the
 * CEO's reset form just said "minimum 12" in a placeholder. People met a
 * refusal the form had not warned them about (audit D-14).
 */

import { t } from "@shared/i18n";

export const PASSWORD_MIN = 12;
export const PASSWORD_MAX = 128;

/** Wording shown next to the field; the same sentence the API returns. */
export function passwordRuleText(): string {
  return t("common.password.rule");
}

/**
 * Returns null when the password is acceptable, otherwise the reason to show.
 *
 * Length counts characters, not code units or bytes: [...p] splits surrogate
 * pairs correctly, so an emoji counts as one, exactly as the server's rune
 * count does.
 */
export function passwordProblem(p: string): string | null {
  const length = [...p].length;
  if (length < PASSWORD_MIN) return t("common.password.tooShort", { min: PASSWORD_MIN });
  if (length > PASSWORD_MAX) return t("common.password.tooLong", { max: PASSWORD_MAX });
  // \p{L} and \p{Nd} are Unicode-wide: any alphabet counts, as on the server.
  if (!/\p{L}/u.test(p)) return t("common.password.needLetter");
  if (!/\p{Nd}/u.test(p)) return t("common.password.needDigit");
  return null;
}
