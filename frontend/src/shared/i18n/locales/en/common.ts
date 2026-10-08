import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.today": "Today",
  "common.yesterday": "Yesterday",
  "common.justNow": "just now",
  "common.minutesAgo": "{count} min ago",
  "common.hoursAgo": "{count} h ago",
  "common.rateLimited": "Too many attempts. Try again a little later",
  "common.rateLimitedSeconds": "Too many attempts. Try again in {count} s",
  "common.rateLimitedMinutes": "Too many attempts. Try again in {count} min",
};
