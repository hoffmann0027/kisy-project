package i18n

// Russian: the source text. Every other catalog has exactly these keys, with
// the same fmt verbs in the same order (i18n_test.go checks it).
// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var ru = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "Проверка недоступна, попробуйте через минуту",
	"auth.captchaFailed":      "Не удалось подтвердить, что вы не робот. Обновите страницу и попробуйте ещё раз",
	"auth.passwordRule":       "пароль: 12–128 символов, минимум одна буква и одна цифра",

	// --- people and chats ---
	"blocks.self":                 "нельзя заблокировать себя",
	"reports.self":                "нельзя пожаловаться на себя",
	"chats.restricted":            "Пользователь ограничил переписку",
	"messages.encryptionRequired": "Личные сообщения отправляются только зашифрованными. Обновите приложение.",

	// --- account ---
	"users.deleteWord":        "УДАЛИТЬ",
	"users.deleteConfirm":     "подтвердите удаление словом %s",
	"users.wrongPassword":     "неверный пароль",
	"users.ownerCannotDelete": "аккаунт владельца удалить нельзя: сначала передайте управление",
	"users.nameTaken":         "Имя занято",
	"users.nameChars":         "Только буквы и одиночные пробелы между словами",
	"users.nameLength":        "Имя: от 2 до 40 символов",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "только CEO может менять уровень группы",
	"groups.sanctionedDelete":   "Сообщество под санкциями модерации — пока они действуют, удалить его нельзя",
	"groups.levelAboveYours":    "нельзя создать группу с уровнем доступа выше вашего",
	"groups.membersByOwnerOnly": "участников добавляет только владелец группы, а в сообщество вступают сами",
	"posts.fileTooLarge":        "файл слишком большой",
	"posts.membersOnly":         "сообщество закрытое: записи видят только участники",
	"posts.noRights":            "у вас нет прав публиковать здесь",
	"posts.notCommunity":        "это группа, а не сообщество",
	"posts.empty":               "пост не может быть пустым",
	"posts.tooLong":             "пост слишком длинный",
	"boards.defaultTitle":       "Доска задач",
	"boards.columnTodo":         "К выполнению",
	"boards.columnDoing":        "В работе",
	"boards.columnDone":         "Готово",

	// --- moderation ---
	"moderation.reasonRequired":     "Укажите причину",
	"moderation.reasonTooLong":      "Причина — не длиннее 1000 символов",
	"moderation.restoreFirst":       "Сообщество удалено — сначала восстановите его",
	"moderation.notDeleted":         "Сообщество не удалено",
	"moderation.sanctionPermanent":  "Эту санкцию нельзя снять",
	"moderation.restoreExpired":     "Срок восстановления истёк",
	"moderation.warn.community":     "Сообществу «%s» вынесено предупреждение (%d из %d): %s",
	"moderation.warn.group":         "Группе «%s» вынесено предупреждение (%d из %d): %s",
	"moderation.mute.community":     "Сообщество «%s» замучено %s — посты не показываются в ленте: %s",
	"moderation.mute.group":         "Группа «%s» замучена %s — посты не показываются в ленте: %s",
	"moderation.delete.community":   "Сообщество «%s» удалено: %s",
	"moderation.delete.group":       "Группа «%s» удалена: %s",
	"moderation.restored.community": "Сообщество «%s» восстановлено",
	"moderation.restored.group":     "Группа «%s» восстановлена",
	"moderation.untilForever":       "бессрочно",
	"moderation.until":              "до %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Публикация постов откроется %s",
	"quarantine.communities":   "Создание сообществ откроется %s",
	"quarantine.newChats":      "Пока можно начинать немного новых переписок в сутки. Ограничение снимется %s",
	"quarantine.upload":        "Новый аккаунт может отправлять файлы поменьше. Ограничение снимется %s",
	"quarantine.other":         "Функция откроется %s",
	"quarantine.inAnHour":      "через час",
	"quarantine.inHours#one":   "через %d час",
	"quarantine.inHours#few":   "через %d часа",
	"quarantine.inHours#many":  "через %d часов",
	"quarantine.inHours#other": "через %d часа",
	"quota.userStorage":        "Место для ваших файлов закончилось: удалите старые вложения или заметки",
	"quota.communityStorage":   "Место для медиа в этом сообществе закончилось",
	"quota.postRate":           "Слишком много постов за час — попробуйте позже",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Отзыв можно оставлять раз в сутки. Следующий — меньше чем через час",
	"feedback.nextInHours":     "Отзыв можно оставлять раз в сутки. Следующий — через %d ч",
	"feedback.replyPushTitle":  "Ответ на ваш отзыв",
	"announcements.dailyLimit": "Лимит на сутки исчерпан: не больше %d рассылок и %d личных уведомлений за 24 часа",
	"releases.pushTitle":       "Вышла версия %s",

	// --- pushes ---
	"push.mentioned":  "Вас упомянули в сообщении",
	"push.newMessage": "Новое сообщение",

	// --- admin ---
	"dashboard.down":          "не отвечает",
	"dashboard.objectStorage": "объектное хранилище",
	"dashboard.inDatabase":    "в базе данных",
	"rating.csv.date":         "Дата",
	"rating.csv.project":      "Проект",
	"rating.csv.task":         "Задача",
	"rating.csv.income":       "Доход",
	"rating.csv.expense":      "Расход",
	"rating.csv.profit":       "Прибыль",
	"rating.csv.author":       "Автор",
	"rating.csv.comment":      "Комментарий",
}
