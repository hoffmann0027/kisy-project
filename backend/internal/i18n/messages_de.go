package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var de = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "Die Prüfung ist gerade nicht verfügbar. Versuch es in einer Minute erneut",
	"auth.captchaFailed":      "Wir konnten nicht bestätigen, dass du kein Roboter bist. Lade die Seite neu und versuch es erneut",
	"auth.passwordRule":       "Passwort: 12–128 Zeichen, mindestens ein Buchstabe und eine Ziffer",

	// --- people and chats ---
	"blocks.self":                 "Du kannst dich nicht selbst blockieren",
	"reports.self":                "Du kannst dich nicht selbst melden",
	"chats.restricted":            "Diese Person hat eingeschränkt, wer ihr schreiben kann",
	"messages.encryptionRequired": "Private Nachrichten werden nur verschlüsselt gesendet. Aktualisiere die App.",

	// --- account ---
	"users.deleteWord":        "LÖSCHEN",
	"users.deleteConfirm":     "Bestätige die Löschung mit dem Wort %s",
	"users.wrongPassword":     "Falsches Passwort",
	"users.ownerCannotDelete": "Das Konto des Inhabers kann nicht gelöscht werden: Übergib zuerst die Verwaltung",
	"users.nameTaken":         "Dieser Name ist bereits vergeben",
	"users.nameChars":         "Nur Buchstaben und einzelne Leerzeichen zwischen Wörtern",
	"users.nameLength":        "Name: 2 bis 40 Zeichen",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "Nur der CEO kann die Stufe einer Gruppe ändern",
	"groups.sanctionedDelete":   "Gegen diese Community laufen Moderationssanktionen – solange sie gelten, kann sie nicht gelöscht werden",
	"groups.levelAboveYours":    "Du kannst keine Gruppe mit einer Zugriffsstufe oberhalb deiner eigenen erstellen",
	"groups.membersByOwnerOnly": "Mitglieder fügt nur der Inhaber der Gruppe hinzu – einer Community tritt man selbst bei",
	"groups.banned":             "Du wurdest aus dieser Community gesperrt",
	"groups.founderStays":       "Der Gründer kann nicht austreten — die Community kann nur gelöscht werden",
	"groups.cannotDiscipline":   "Entfernen und sperren kann nur, wer in der Gruppe eine höhere Rolle hat",
	"posts.fileTooLarge":        "Die Datei ist zu groß",
	"posts.membersOnly":         "Die Community ist geschlossen: Beiträge sehen nur Mitglieder",
	"posts.noRights":            "Du darfst hier nichts veröffentlichen",
	"posts.notCommunity":        "Das ist eine Gruppe, keine Community",
	"posts.empty":               "Ein Beitrag darf nicht leer sein",
	"posts.tooLong":             "Der Beitrag ist zu lang",
	"boards.defaultTitle":       "Aufgabenboard",
	"boards.columnTodo":         "Zu erledigen",
	"boards.columnDoing":        "In Arbeit",
	"boards.columnDone":         "Erledigt",

	// --- moderation ---
	"moderation.reasonRequired":     "Gib einen Grund an",
	"moderation.reasonTooLong":      "Der Grund darf höchstens 1000 Zeichen lang sein",
	"moderation.restoreFirst":       "Die Community wurde gelöscht – stelle sie zuerst wieder her",
	"moderation.notDeleted":         "Die Community ist nicht gelöscht",
	"moderation.sanctionPermanent":  "Diese Sanktion kann nicht aufgehoben werden",
	"moderation.restoreExpired":     "Die Frist für die Wiederherstellung ist abgelaufen",
	"moderation.warn.community":     "Die Community „%s“ hat eine Verwarnung erhalten (%d von %d): %s",
	"moderation.warn.group":         "Die Gruppe „%s“ hat eine Verwarnung erhalten (%d von %d): %s",
	"moderation.mute.community":     "Die Community „%s“ ist %s stummgeschaltet – ihre Beiträge erscheinen nicht im Feed: %s",
	"moderation.mute.group":         "Die Gruppe „%s“ ist %s stummgeschaltet – ihre Beiträge erscheinen nicht im Feed: %s",
	"moderation.delete.community":   "Die Community „%s“ wurde gelöscht: %s",
	"moderation.delete.group":       "Die Gruppe „%s“ wurde gelöscht: %s",
	"moderation.restored.community": "Die Community „%s“ wurde wiederhergestellt",
	"moderation.restored.group":     "Die Gruppe „%s“ wurde wiederhergestellt",
	"moderation.untilForever":       "unbefristet",
	"moderation.until":              "bis %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Das Veröffentlichen von Beiträgen wird %s freigeschaltet",
	"quarantine.communities":   "Das Erstellen von Communitys wird %s freigeschaltet",
	"quarantine.newChats":      "Vorerst kannst du nur wenige neue Unterhaltungen pro Tag beginnen. Die Beschränkung endet %s",
	"quarantine.upload":        "Neue Konten können vorerst nur kleinere Dateien senden. Die Beschränkung endet %s",
	"quarantine.other":         "Diese Funktion wird %s freigeschaltet",
	"quarantine.inAnHour":      "in einer Stunde",
	"quarantine.inHours#one":   "in %d Stunde",
	"quarantine.inHours#other": "in %d Stunden",
	"quota.userStorage":        "Der Speicherplatz für deine Dateien ist voll: Lösche alte Anhänge oder Notizen",
	"quota.communityStorage":   "Der Speicherplatz für Medien in dieser Community ist voll",
	"quota.postRate":           "Zu viele Beiträge in einer Stunde – versuch es später erneut",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Du kannst einmal pro Tag Feedback geben. Das nächste Mal in weniger als einer Stunde",
	"feedback.nextInHours":     "Du kannst einmal pro Tag Feedback geben. Das nächste Mal in %d Std.",
	"feedback.replyPushTitle":  "Antwort auf dein Feedback",
	"announcements.dailyLimit": "Tageslimit erreicht: höchstens %d Rundnachrichten und %d persönliche Mitteilungen in 24 Stunden",
	"releases.pushTitle":       "Version %s ist da",

	// --- pushes ---
	"push.mentioned":  "Du wurdest in einer Nachricht erwähnt",
	"push.newMessage": "Neue Nachricht",

	// --- admin ---
	"dashboard.down":          "antwortet nicht",
	"dashboard.objectStorage": "Objektspeicher",
	"dashboard.inDatabase":    "in der Datenbank",
	"rating.csv.date":         "Datum",
	"rating.csv.project":      "Projekt",
	"rating.csv.task":         "Aufgabe",
	"rating.csv.income":       "Einnahmen",
	"rating.csv.expense":      "Ausgaben",
	"rating.csv.profit":       "Gewinn",
	"rating.csv.author":       "Autor",
	"rating.csv.comment":      "Kommentar",
}

func init() { catalogs["de"] = de }
