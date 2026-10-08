package i18n

// Ukrainian. Same keys as messages_ru.go, with the same fmt verbs in the same
// order (i18n_test.go checks it). "Група" and "Спільнота" are both feminine.
// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var uk = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "Перевірка недоступна, спробуйте за хвилину",
	"auth.captchaFailed":      "Не вдалося підтвердити, що ви не робот. Оновіть сторінку й спробуйте ще раз",
	"auth.passwordRule":       "пароль: 12–128 символів, щонайменше одна літера й одна цифра",

	// --- people and chats ---
	"blocks.self":                 "не можна заблокувати себе",
	"reports.self":                "не можна поскаржитися на себе",
	"chats.restricted":            "Користувач обмежив листування",
	"messages.encryptionRequired": "Особисті повідомлення надсилаються лише зашифрованими. Оновіть застосунок.",

	// --- account ---
	"users.deleteWord":        "ВИДАЛИТИ",
	"users.deleteConfirm":     "підтвердьте видалення словом %s",
	"users.wrongPassword":     "неправильний пароль",
	"users.ownerCannotDelete": "акаунт власника видалити не можна: спершу передайте керування",
	"users.nameTaken":         "Ім’я зайняте",
	"users.nameChars":         "Лише літери й одинарні пробіли між словами",
	"users.nameLength":        "Ім’я: від 2 до 40 символів",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "лише CEO може змінювати рівень групи",
	"groups.sanctionedDelete":   "Спільнота під санкціями модерації — поки вони діють, видалити її не можна",
	"groups.levelAboveYours":    "не можна створити групу з рівнем доступу, вищим за ваш",
	"groups.membersByOwnerOnly": "учасників додає лише власник групи, а до спільноти приєднуються самі",
	"groups.banned":             "Вас заблоковано в цій спільноті",
	"groups.founderStays":       "Засновник не може вийти зі спільноти — її можна лише видалити",
	"groups.cannotDiscipline":   "Виключати й блокувати може лише той, хто має вищу роль у групі",
	"posts.fileTooLarge":        "файл завеликий",
	"posts.membersOnly":         "спільнота закрита: дописи бачать лише учасники",
	"posts.noRights":            "у вас немає прав публікувати тут",
	"posts.notCommunity":        "це група, а не спільнота",
	"posts.empty":               "допис не може бути порожнім",
	"posts.tooLong":             "допис задовгий",
	"boards.defaultTitle":       "Дошка завдань",
	"boards.columnTodo":         "До виконання",
	"boards.columnDoing":        "У роботі",
	"boards.columnDone":         "Готово",

	// --- moderation ---
	"moderation.reasonRequired":     "Вкажіть причину",
	"moderation.reasonTooLong":      "Причина — не довша за 1000 символів",
	"moderation.restoreFirst":       "Спільноту видалено — спершу відновіть її",
	"moderation.notDeleted":         "Спільноту не видалено",
	"moderation.sanctionPermanent":  "Цю санкцію не можна зняти",
	"moderation.restoreExpired":     "Строк відновлення минув",
	"moderation.warn.community":     "Спільноті «%s» винесено попередження (%d з %d): %s",
	"moderation.warn.group":         "Групі «%s» винесено попередження (%d з %d): %s",
	"moderation.mute.community":     "Спільноту «%s» заглушено %s — її дописи не показуються у стрічці: %s",
	"moderation.mute.group":         "Групу «%s» заглушено %s — її дописи не показуються у стрічці: %s",
	"moderation.delete.community":   "Спільноту «%s» видалено: %s",
	"moderation.delete.group":       "Групу «%s» видалено: %s",
	"moderation.restored.community": "Спільноту «%s» відновлено",
	"moderation.restored.group":     "Групу «%s» відновлено",
	"moderation.untilForever":       "безстроково",
	"moderation.until":              "до %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Публікація дописів стане доступною %s",
	"quarantine.communities":   "Створення спільнот стане доступним %s",
	"quarantine.newChats":      "Поки що можна починати небагато нових розмов на добу. Обмеження буде знято %s",
	"quarantine.upload":        "Новий акаунт може надсилати лише менші файли. Обмеження буде знято %s",
	"quarantine.other":         "Функція стане доступною %s",
	"quarantine.inAnHour":      "через годину",
	"quarantine.inHours#one":   "через %d годину",
	"quarantine.inHours#few":   "через %d години",
	"quarantine.inHours#many":  "через %d годин",
	"quarantine.inHours#other": "через %d години",
	"quota.userStorage":        "Місце для ваших файлів закінчилося: видаліть старі вкладення або нотатки",
	"quota.communityStorage":   "Місце для медіа в цій спільноті закінчилося",
	"quota.postRate":           "Забагато дописів за годину — спробуйте пізніше",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Відгук можна залишати раз на добу. Наступний — менш ніж через годину",
	"feedback.nextInHours":     "Відгук можна залишати раз на добу. Наступний — через %d год",
	"feedback.replyPushTitle":  "Відповідь на ваш відгук",
	"announcements.dailyLimit": "Ліміт на добу вичерпано: не більше ніж %d розсилок і %d особистих сповіщень за 24 години",
	"releases.pushTitle":       "Вийшла версія %s",

	// --- pushes ---
	"push.mentioned":  "Вас згадали в повідомленні",
	"push.newMessage": "Нове повідомлення",

	// --- admin ---
	"dashboard.down":          "не відповідає",
	"dashboard.objectStorage": "об’єктне сховище",
	"dashboard.inDatabase":    "у базі даних",
	"rating.csv.date":         "Дата",
	"rating.csv.project":      "Проєкт",
	"rating.csv.task":         "Завдання",
	"rating.csv.income":       "Дохід",
	"rating.csv.expense":      "Витрата",
	"rating.csv.profit":       "Прибуток",
	"rating.csv.author":       "Автор",
	"rating.csv.comment":      "Коментар",
}

func init() { catalogs["uk"] = uk }
