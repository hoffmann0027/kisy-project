package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var pl = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "Weryfikacja jest chwilowo niedostępna, spróbuj za minutę",
	"auth.captchaFailed":      "Nie udało się potwierdzić, że nie jesteś robotem. Odśwież stronę i spróbuj ponownie",
	"auth.passwordRule":       "hasło: 12–128 znaków, co najmniej jedna litera i jedna cyfra",

	// --- people and chats ---
	"blocks.self":                 "nie można zablokować siebie",
	"reports.self":                "nie można zgłosić siebie",
	"chats.restricted":            "Ta osoba ograniczyła, kto może do niej pisać",
	"messages.encryptionRequired": "Wiadomości prywatne można wysyłać tylko zaszyfrowane. Zaktualizuj aplikację.",

	// --- account ---
	"users.deleteWord":        "USUŃ",
	"users.deleteConfirm":     "potwierdź usunięcie słowem %s",
	"users.wrongPassword":     "nieprawidłowe hasło",
	"users.ownerCannotDelete": "nie można usunąć konta właściciela: najpierw przekaż zarządzanie",
	"users.nameTaken":         "To imię jest już zajęte",
	"users.nameChars":         "Tylko litery i pojedyncze spacje między słowami",
	"users.nameLength":        "Imię: od 2 do 40 znaków",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "tylko CEO może zmieniać poziom grupy",
	"groups.sanctionedDelete":   "Na społeczność nałożono sankcje moderacyjne — dopóki obowiązują, nie można jej usunąć",
	"groups.levelAboveYours":    "nie możesz utworzyć grupy z poziomem dostępu wyższym niż twój",
	"groups.membersByOwnerOnly": "członków dodaje tylko właściciel grupy, a do społeczności dołącza się samodzielnie",
	"groups.banned":             "Zostałeś zablokowany w tej społeczności",
	"groups.founderStays":       "Założyciel nie może odejść — społeczność można tylko usunąć",
	"groups.cannotDiscipline":   "Usuwać i blokować może tylko ktoś z wyższą rolą w grupie",
	"posts.fileTooLarge":        "plik jest za duży",
	"posts.membersOnly":         "to zamknięta społeczność: posty widzą tylko jej członkowie",
	"posts.noRights":            "nie masz uprawnień do publikowania tutaj",
	"posts.notCommunity":        "to jest grupa, a nie społeczność",
	"posts.empty":               "post nie może być pusty",
	"posts.tooLong":             "post jest za długi",
	"boards.defaultTitle":       "Tablica zadań",
	"boards.columnTodo":         "Do zrobienia",
	"boards.columnDoing":        "W toku",
	"boards.columnDone":         "Gotowe",

	// --- moderation ---
	"moderation.reasonRequired":     "Podaj powód",
	"moderation.reasonTooLong":      "Powód może mieć maksymalnie 1000 znaków",
	"moderation.restoreFirst":       "Społeczność została usunięta — najpierw ją przywróć",
	"moderation.notDeleted":         "Społeczność nie jest usunięta",
	"moderation.sanctionPermanent":  "Tej sankcji nie można zdjąć",
	"moderation.restoreExpired":     "Termin na przywrócenie upłynął",
	"moderation.warn.community":     "Społeczność „%s” otrzymała ostrzeżenie (%d z %d): %s",
	"moderation.warn.group":         "Grupa „%s” otrzymała ostrzeżenie (%d z %d): %s",
	"moderation.mute.community":     "Społeczność „%s” jest wyciszona %s — jej posty nie są pokazywane w Aktualnościach: %s",
	"moderation.mute.group":         "Grupa „%s” jest wyciszona %s — jej posty nie są pokazywane w Aktualnościach: %s",
	"moderation.delete.community":   "Społeczność „%s” została usunięta: %s",
	"moderation.delete.group":       "Grupa „%s” została usunięta: %s",
	"moderation.restored.community": "Społeczność „%s” została przywrócona",
	"moderation.restored.group":     "Grupa „%s” została przywrócona",
	"moderation.untilForever":       "bezterminowo",
	"moderation.until":              "do %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Publikowanie postów będzie dostępne %s",
	"quarantine.communities":   "Tworzenie społeczności będzie dostępne %s",
	"quarantine.newChats":      "Na razie możesz rozpoczynać tylko kilka nowych rozmów dziennie. Limit zniknie %s",
	"quarantine.upload":        "Nowe konto może wysyłać tylko mniejsze pliki. Limit zniknie %s",
	"quarantine.other":         "Ta funkcja będzie dostępna %s",
	"quarantine.inAnHour":      "za godzinę",
	"quarantine.inHours#one":   "za %d godzinę",
	"quarantine.inHours#few":   "za %d godziny",
	"quarantine.inHours#many":  "za %d godzin",
	"quarantine.inHours#other": "za %d godziny",
	"quota.userStorage":        "Skończyło się miejsce na twoje pliki: usuń stare załączniki lub notatki",
	"quota.communityStorage":   "W tej społeczności skończyło się miejsce na multimedia",
	"quota.postRate":           "Za dużo postów w ciągu godziny — spróbuj później",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Opinię można wysłać raz na dobę. Następną wyślesz za mniej niż godzinę",
	"feedback.nextInHours":     "Opinię można wysłać raz na dobę. Następną wyślesz za %d godz.",
	"feedback.replyPushTitle":  "Odpowiedź na twoją opinię",
	"announcements.dailyLimit": "Wyczerpano limit na 24 godziny: komunikaty do wielu osób — maks. %d, powiadomienia osobiste — maks. %d",
	"releases.pushTitle":       "Dostępna jest wersja %s",

	// --- pushes ---
	"push.mentioned":  "Wspomniano o tobie w wiadomości",
	"push.newMessage": "Nowa wiadomość",

	// --- admin ---
	"dashboard.down":          "nie odpowiada",
	"dashboard.objectStorage": "magazyn obiektów",
	"dashboard.inDatabase":    "w bazie danych",
	"rating.memberCannotSee":  "Ten użytkownik nie widzi projektu: jego poziom jest niższy niż poziom dostępu projektu",
	"rating.csv.date":         "Data",
	"rating.csv.project":      "Projekt",
	"rating.csv.task":         "Zadanie",
	"rating.csv.income":       "Przychód",
	"rating.csv.expense":      "Wydatek",
	"rating.csv.profit":       "Zysk",
	"rating.csv.author":       "Autor",
	"rating.csv.comment":      "Komentarz",
}

func init() { catalogs["pl"] = pl }
