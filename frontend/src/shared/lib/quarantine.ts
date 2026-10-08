import type { Quarantine } from "@shared/api/types";
import { t } from "@shared/i18n";

// What a freshly self-registered account is held back from for its first hours
// (backend: internal/quarantine). The server enforces it; these helpers only
// let the screens say so up front instead of letting someone write a post and
// lose it on submit.

/** "через 18 часов" — the same wording the server sends. */
export function opensIn(hoursLeft: number): string {
  const h = Math.max(1, Math.ceil(hoursLeft));
  if (h === 1) return t("common.quarantine.inAnHour");
  return t("common.quarantine.inHours", { count: h });
}

/** The notice shown in place of a feature the hold closes. */
export function heldBackNotice(q: Quarantine | null, feature: string): string | null {
  if (!q) return null;
  return t("common.quarantine.heldBack", { feature, when: opensIn(q.hoursLeft) });
}

/** Megabytes, for a message about the file ceiling. */
function mb(bytes: number): string {
  return t("common.units.mb", { value: Math.round((bytes / (1024 * 1024)) * 10) / 10 });
}

/**
 * The refusal for a file the hold is too small for, or null when it fits.
 * Checked before the upload starts: sending a 30 MB video over a phone
 * connection only to be refused at the end is the worst version of this.
 */
export function fileTooLargeForNewAccount(q: Quarantine | null, file: { size: number }): string | null {
  if (!q || q.maxUploadBytes <= 0 || file.size <= q.maxUploadBytes) return null;
  return t("common.quarantine.fileTooLarge", { size: mb(q.maxUploadBytes), when: opensIn(q.hoursLeft) });
}
