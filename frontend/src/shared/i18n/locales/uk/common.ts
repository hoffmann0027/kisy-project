import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Видалений акаунт",
  "common.today": "Сьогодні",
  "common.yesterday": "Учора",
  "common.justNow": "щойно",
  "common.minutesAgo": "{count} хв тому",
  "common.hoursAgo": "{count} год тому",
  "common.rateLimited": "Забагато спроб. Спробуйте трохи згодом",
  "common.rateLimitedSeconds": "Забагато спроб. Спробуйте через {count} с",
  "common.rateLimitedMinutes": "Забагато спроб. Спробуйте через {count} хв",

  "common.loading": "Завантаження",
  "common.modal.close": "Закрити",
  "common.code.copy": "Копіювати",

  "common.verified.account": "Підтверджений акаунт",
  "common.verified.community": "Підтверджена спільнота",

  "common.media.resetZoom": "Скинути масштаб (0)",
  "common.media.download": "Завантажити",
  "common.media.close": "Закрити (Esc)",
  "common.media.previous": "Попереднє",
  "common.media.next": "Наступне",

  "common.emoji.picker": "Вибір емодзі",
  "common.emoji.search": "Пошук емодзі",
  "common.emoji.noResults": "Нічого не знайдено",
  "common.emoji.recent": "Нещодавні",
  "common.emoji.smileys": "Смайли",
  "common.emoji.gestures": "Жести",
  "common.emoji.hearts": "Серця",
  "common.emoji.objects": "Об’єкти",

  "common.nav.messages": "Повідомлення",
  "common.nav.communities": "Спільноти",
  "common.nav.rating": "Рейтинг",
  "common.nav.feed": "Стрічка",

  "common.push.channelName": "Повідомлення",
  "common.push.channelDescription": "Нові повідомлення та згадки",

  "common.displayName.taken": "Ім’я зайняте",
  "common.displayName.letters": "Лише літери й одинарні пробіли між словами",
  "common.displayName.length": "Ім’я: від 2 до 40 символів",

  "common.password.rule": "12–128 символів, щонайменше одна літера й одна цифра",
  "common.password.tooShort": "Щонайменше {min} символів",
  "common.password.tooLong": "Не більше ніж {max} символів",
  "common.password.needLetter": "Потрібна хоча б одна літера",
  "common.password.needDigit": "Потрібна хоча б одна цифра",

  "common.quarantine.inAnHour": "через годину",
  "common.quarantine.inHours": {
    one: "через {count} годину",
    few: "через {count} години",
    many: "через {count} годин",
    other: "через {count} години",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge": "Новий акаунт може надсилати файли до {size}. Обмеження буде знято {when}",

  "common.units.bytes": "{value} Б",
  "common.units.kb": "{value} КБ",
  "common.units.mb": "{value} МБ",
  "common.units.gb": "{value} ГБ",

  "common.duration.oneDay": "1 день",
  "common.duration.days": "{count} дн.",
  "common.duration.oneHour": "1 година",
  "common.duration.hours": "{count} год",
  "common.duration.minutes": "{count} хв",
  "common.duration.seconds": "{count} с",

  "common.group.group": "Група",
  "common.group.community": "Спільнота",
  "common.group.openCommunity": "Відкрита спільнота",
  "common.group.fromRole": "{kind} · від {role} і вище",

  "common.session.refreshFailed": "Не вдалося оновити сесію",
  "common.consent.saveFailed": "Не вдалося зберегти згоду. Спробуйте ще раз",

  "common.offline.title": "Немає зв’язку із сервером",
  "common.offline.body": "Ви залишаєтеся в акаунті — застосунок підключиться сам, щойно з’явиться мережа.",
  "common.offline.retry": "Повторити",

  "common.e2ee.deviceNotInChat":
    "Цей пристрій ще не підключено до захищеного чату. Його підключить співрозмовник або інший ваш пристрій, щойно буде в мережі, — тоді надішліть ще раз.",
  "common.e2ee.peerNoDevices": "Співрозмовник ще не входив у KISY з підтримкою шифрування — повідомлення не надіслано.",
  "common.e2ee.peerKeysExhausted":
    "У співрозмовника закінчилися ключі шифрування — повідомлення не надіслано. Попросіть його відкрити KISY і повторіть.",
  "common.e2ee.unavailable":
    "Шифрування на цьому пристрої не запустилося — повідомлення не надіслано. Перезапустіть застосунок і повторіть.",
  "common.e2ee.peerUnknown": "Не вдалося визначити співрозмовника — повідомлення не надіслано.",
  "common.e2ee.encryptFailed": "Не вдалося зашифрувати повідомлення — його не надіслано. Спробуйте ще раз.",

  "common.forward.nothingToForward": "Немає повідомлень, які можна переслати",
};
