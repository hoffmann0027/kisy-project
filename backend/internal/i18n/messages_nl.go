package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var nl = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "De controle is niet beschikbaar. Probeer het over een minuut opnieuw",
	"auth.captchaFailed":      "Kan niet bevestigen dat je geen robot bent. Vernieuw de pagina en probeer het opnieuw",
	"auth.passwordRule":       "wachtwoord: 12–128 tekens, minstens één letter en één cijfer",

	// --- people and chats ---
	"blocks.self":                 "Je kunt jezelf niet blokkeren",
	"reports.self":                "Je kunt jezelf niet rapporteren",
	"chats.restricted":            "Deze gebruiker heeft beperkt wie berichten mag sturen",
	"messages.encryptionRequired": "Privéberichten kunnen alleen versleuteld worden verstuurd. Werk de app bij.",

	// --- account ---
	"users.deleteWord":        "VERWIJDEREN",
	"users.deleteConfirm":     "bevestig het verwijderen met het woord %s",
	"users.wrongPassword":     "onjuist wachtwoord",
	"users.ownerCannotDelete": "Het account van de eigenaar kan niet worden verwijderd: draag eerst het beheer over",
	"users.nameTaken":         "Deze naam is al bezet",
	"users.nameChars":         "Alleen letters en enkele spaties tussen woorden",
	"users.nameLength":        "Naam: 2 tot 40 tekens",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "Alleen de CEO kan het niveau van een groep wijzigen",
	"groups.sanctionedDelete":   "Op deze community rusten moderatiesancties — zolang die gelden, kan deze niet worden verwijderd",
	"groups.levelAboveYours":    "Je kunt geen groep maken met een toegangsniveau boven je eigen niveau",
	"groups.membersByOwnerOnly": "Alleen de eigenaar van een groep voegt leden toe; bij een community word je zelf lid",
	"groups.banned":             "Je bent geblokkeerd in deze community",
	"groups.founderStays":       "De oprichter kan niet vertrekken — de community kan alleen worden verwijderd",
	"groups.cannotDiscipline":   "Alleen iemand met een hogere rol in de groep kan verwijderen of blokkeren",
	"posts.fileTooLarge":        "Het bestand is te groot",
	"posts.membersOnly":         "Deze community is besloten: alleen leden zien de posts",
	"posts.noRights":            "Je hebt geen rechten om hier te publiceren",
	"posts.notCommunity":        "Dit is een groep, geen community",
	"posts.empty":               "Een post mag niet leeg zijn",
	"posts.tooLong":             "De post is te lang",
	"boards.defaultTitle":       "Takenbord",
	"boards.columnTodo":         "Te doen",
	"boards.columnDoing":        "Bezig",
	"boards.columnDone":         "Klaar",

	// --- moderation ---
	"moderation.reasonRequired":     "Geef een reden op",
	"moderation.reasonTooLong":      "De reden mag maximaal 1000 tekens lang zijn",
	"moderation.restoreFirst":       "De community is verwijderd — herstel deze eerst",
	"moderation.notDeleted":         "De community is niet verwijderd",
	"moderation.sanctionPermanent":  "Deze sanctie kan niet worden opgeheven",
	"moderation.restoreExpired":     "De hersteltermijn is verlopen",
	"moderation.warn.community":     "Community “%s” heeft een waarschuwing gekregen (%d van %d): %s",
	"moderation.warn.group":         "Groep “%s” heeft een waarschuwing gekregen (%d van %d): %s",
	"moderation.mute.community":     "Community “%s” is gedempt %s — posts worden niet in de feed getoond: %s",
	"moderation.mute.group":         "Groep “%s” is gedempt %s — posts worden niet in de feed getoond: %s",
	"moderation.delete.community":   "Community “%s” is verwijderd: %s",
	"moderation.delete.group":       "Groep “%s” is verwijderd: %s",
	"moderation.restored.community": "Community “%s” is hersteld",
	"moderation.restored.group":     "Groep “%s” is hersteld",
	"moderation.untilForever":       "voor onbepaalde tijd",
	"moderation.until":              "tot %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Posts publiceren kan pas %s",
	"quarantine.communities":   "Community's maken kan pas %s",
	"quarantine.newChats":      "Voorlopig kun je maar een paar nieuwe gesprekken per dag beginnen. Deze beperking vervalt %s",
	"quarantine.upload":        "Een nieuw account kan alleen kleinere bestanden versturen. Deze beperking vervalt %s",
	"quarantine.other":         "Deze functie is pas beschikbaar %s",
	"quarantine.inAnHour":      "over een uur",
	"quarantine.inHours#one":   "over %d uur",
	"quarantine.inHours#other": "over %d uur",
	"quota.userStorage":        "De ruimte voor je bestanden is op: verwijder oude bijlagen of notities",
	"quota.communityStorage":   "De ruimte voor media in deze community is op",
	"quota.postRate":           "Te veel posts in een uur — probeer het later opnieuw",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Je kunt één keer per dag feedback geven. De volgende kan binnen een uur",
	"feedback.nextInHours":     "Je kunt één keer per dag feedback geven. De volgende kan over %d uur",
	"feedback.replyPushTitle":  "Antwoord op je feedback",
	"announcements.dailyLimit": "Daglimiet bereikt: maximaal %d algemene en %d persoonlijke mededelingen per 24 uur",
	"releases.pushTitle":       "Versie %s is uit",

	// --- pushes ---
	"push.mentioned":  "Je bent vermeld in een bericht",
	"push.newMessage": "Nieuw bericht",

	// --- admin ---
	"dashboard.down":          "reageert niet",
	"dashboard.objectStorage": "objectopslag",
	"dashboard.inDatabase":    "in de database",
	"rating.memberCannotSee":  "Deze gebruiker ziet het project niet: zijn niveau ligt onder het toegangsniveau van het project",
	"rating.csv.date":         "Datum",
	"rating.csv.project":      "Project",
	"rating.csv.task":         "Taak",
	"rating.csv.income":       "Inkomsten",
	"rating.csv.expense":      "Uitgaven",
	"rating.csv.profit":       "Winst",
	"rating.csv.author":       "Auteur",
	"rating.csv.comment":      "Opmerking",
}

func init() { catalogs["nl"] = nl }
