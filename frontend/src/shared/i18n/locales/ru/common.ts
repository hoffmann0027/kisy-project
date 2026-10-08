// Russian — the source text. Area: shared/ (ui, lib, api), app/, entities/.
import type { Dict } from "../../types";

export const common = {
  "common.today": "Сегодня",
  "common.yesterday": "Вчера",
  "common.justNow": "только что",
  "common.minutesAgo": "{count} мин назад",
  "common.hoursAgo": "{count} ч назад",
  "common.rateLimited": "Слишком часто. Попробуйте чуть позже",
  "common.rateLimitedSeconds": "Слишком часто. Попробуйте через {count} с",
  "common.rateLimitedMinutes": "Слишком часто. Попробуйте через {count} мин",
} satisfies Dict;
