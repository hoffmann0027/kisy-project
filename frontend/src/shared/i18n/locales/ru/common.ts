// Russian — the source text. Area: shared/ (ui, lib, api), app/, entities/.
import type { Dict } from "../../types";

export const common = {
  "common.deletedAccount": "Удалённый аккаунт",
  "common.today": "Сегодня",
  "common.yesterday": "Вчера",
  "common.justNow": "только что",
  "common.minutesAgo": "{count} мин назад",
  "common.hoursAgo": "{count} ч назад",
  "common.rateLimited": "Слишком часто. Попробуйте чуть позже",
  "common.rateLimitedSeconds": "Слишком часто. Попробуйте через {count} с",
  "common.rateLimitedMinutes": "Слишком часто. Попробуйте через {count} мин",

  "common.loading": "Загрузка",
  "common.modal.close": "Закрыть",
  "common.code.copy": "Копировать",

  "common.verified.account": "Подтверждённый аккаунт",
  "common.verified.community": "Подтверждённое сообщество",

  "common.media.resetZoom": "Вернуть масштаб (0)",
  "common.media.download": "Скачать",
  "common.media.close": "Закрыть (Esc)",
  "common.media.previous": "Предыдущее",
  "common.media.next": "Следующее",

  "common.emoji.picker": "Выбор эмодзи",
  "common.emoji.search": "Поиск эмодзи",
  "common.emoji.noResults": "Ничего не найдено",
  "common.emoji.recent": "Недавние",
  "common.emoji.smileys": "Смайлы",
  "common.emoji.gestures": "Жесты",
  "common.emoji.hearts": "Сердца",
  "common.emoji.objects": "Объекты",

  "common.nav.messages": "Сообщения",
  "common.nav.communities": "Сообщества",
  "common.nav.rating": "Рейтинг",
  "common.nav.feed": "Лента",

  "common.push.channelName": "Сообщения",
  "common.push.channelDescription": "Новые сообщения и упоминания",

  "common.displayName.taken": "Имя занято",
  "common.displayName.letters": "Только буквы и одиночные пробелы между словами",
  "common.displayName.length": "Имя: от 2 до 40 символов",

  "common.password.rule": "12–128 символов, минимум одна буква и одна цифра",
  "common.password.tooShort": "Минимум {min} символов",
  "common.password.tooLong": "Не более {max} символов",
  "common.password.needLetter": "Нужна хотя бы одна буква",
  "common.password.needDigit": "Нужна хотя бы одна цифра",

  "common.quarantine.inAnHour": "через час",
  "common.quarantine.inHours": {
    one: "через {count} час",
    few: "через {count} часа",
    many: "через {count} часов",
    other: "через {count} часа",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge": "Новый аккаунт может отправлять файлы до {size}. Ограничение снимется {when}",

  "common.units.bytes": "{value} Б",
  "common.units.kb": "{value} КБ",
  "common.units.mb": "{value} МБ",
  "common.units.gb": "{value} ГБ",

  "common.duration.oneDay": "1 день",
  "common.duration.days": "{count} дн.",
  "common.duration.oneHour": "1 час",
  "common.duration.hours": "{count} ч.",
  "common.duration.minutes": "{count} мин",
  "common.duration.seconds": "{count} сек",

  "common.group.group": "Группа",
  "common.group.community": "Сообщество",
  "common.group.openCommunity": "Открытое сообщество",
  "common.group.fromRole": "{kind} · от {role} и выше",

  "common.session.refreshFailed": "Не удалось обновить сессию",
  "common.consent.saveFailed": "Не удалось сохранить согласие. Попробуйте ещё раз",

  "common.offline.title": "Нет связи с сервером",
  "common.offline.body": "Вы остаётесь в аккаунте — приложение подключится само, как только появится сеть.",
  "common.offline.retry": "Повторить",

  "common.e2ee.deviceNotInChat":
    "Это устройство ещё не подключено к защищённому чату. Его подключит собеседник или другое ваше устройство, как только окажется в сети, — тогда отправьте снова.",
  "common.e2ee.peerNoDevices": "Собеседник ещё не входил в KISY с поддержкой шифрования — сообщение не отправлено.",
  "common.e2ee.peerKeysExhausted":
    "У собеседника закончились ключи шифрования — сообщение не отправлено. Попросите его открыть KISY и повторите.",
  "common.e2ee.unavailable":
    "Шифрование на этом устройстве не запустилось — сообщение не отправлено. Перезапустите приложение и повторите.",
  "common.e2ee.peerUnknown": "Не удалось определить собеседника — сообщение не отправлено.",
  "common.e2ee.encryptFailed": "Не удалось зашифровать сообщение — оно не отправлено. Повторите попытку.",

  "common.forward.nothingToForward": "Нет сообщений, доступных для пересылки",
} satisfies Dict;
