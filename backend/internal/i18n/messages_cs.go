package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var cs = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "Ověření je teď nedostupné, zkus to za minutu",
	"auth.captchaFailed":      "Nepodařilo se ověřit, že nejsi robot. Obnov stránku a zkus to znovu",
	"auth.passwordRule":       "heslo: 12–128 znaků, alespoň jedno písmeno a jedna číslice",

	// --- people and chats ---
	"blocks.self":                 "Nemůžeš zablokovat vlastní účet",
	"reports.self":                "Nemůžeš nahlásit vlastní účet",
	"chats.restricted":            "Uživatel omezil, kdo mu může psát",
	"messages.encryptionRequired": "Soukromé zprávy lze posílat jen šifrované. Aktualizuj aplikaci.",

	// --- account ---
	"users.deleteWord":        "SMAZAT",
	"users.deleteConfirm":     "potvrď smazání napsáním slova %s",
	"users.wrongPassword":     "nesprávné heslo",
	"users.ownerCannotDelete": "Účet vlastníka nelze smazat: nejdřív předej správu",
	"users.nameTaken":         "Toto jméno je obsazené",
	"users.nameChars":         "Jen písmena, mezi slovy vždy jedna mezera",
	"users.nameLength":        "Jméno: 2 až 40 znaků",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "Úroveň skupiny může měnit jen CEO",
	"groups.sanctionedDelete":   "Komunita je pod sankcemi moderace — dokud platí, nelze ji smazat",
	"groups.levelAboveYours":    "Nelze vytvořit skupinu s vyšší úrovní přístupu, než máš ty",
	"groups.membersByOwnerOnly": "Členy do skupiny přidává jen vlastník, do komunity se vstupuje samostatně",
	"posts.fileTooLarge":        "Soubor je příliš velký",
	"posts.membersOnly":         "Komunita je uzavřená: příspěvky vidí jen členové",
	"posts.noRights":            "Nemáš oprávnění tu publikovat",
	"posts.notCommunity":        "Toto je skupina, ne komunita",
	"posts.empty":               "Příspěvek nemůže být prázdný",
	"posts.tooLong":             "Příspěvek je příliš dlouhý",
	"boards.defaultTitle":       "Nástěnka úkolů",
	"boards.columnTodo":         "K řešení",
	"boards.columnDoing":        "Probíhá",
	"boards.columnDone":         "Hotovo",

	// --- moderation ---
	"moderation.reasonRequired":     "Uveď důvod",
	"moderation.reasonTooLong":      "Důvod může mít nejvýše 1000 znaků",
	"moderation.restoreFirst":       "Komunita je smazaná — nejdřív ji obnov",
	"moderation.notDeleted":         "Komunita není smazaná",
	"moderation.sanctionPermanent":  "Tuto sankci nelze zrušit",
	"moderation.restoreExpired":     "Lhůta pro obnovení vypršela",
	"moderation.warn.community":     "Komunita „%s“ dostala varování (%d z %d): %s",
	"moderation.warn.group":         "Skupina „%s“ dostala varování (%d z %d): %s",
	"moderation.mute.community":     "Komunita „%s“ je ztlumena %s — její příspěvky se nezobrazují ve feedu: %s",
	"moderation.mute.group":         "Skupina „%s“ je ztlumena %s — její příspěvky se nezobrazují ve feedu: %s",
	"moderation.delete.community":   "Komunita „%s“ byla smazána: %s",
	"moderation.delete.group":       "Skupina „%s“ byla smazána: %s",
	"moderation.restored.community": "Komunita „%s“ byla obnovena",
	"moderation.restored.group":     "Skupina „%s“ byla obnovena",
	"moderation.untilForever":       "natrvalo",
	"moderation.until":              "do %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Publikování příspěvků se odemkne %s",
	"quarantine.communities":   "Vytváření komunit se odemkne %s",
	"quarantine.newChats":      "Zatím můžeš denně začít jen pár nových konverzací. Omezení skončí %s",
	"quarantine.upload":        "Nový účet může posílat jen menší soubory. Omezení skončí %s",
	"quarantine.other":         "Funkce se odemkne %s",
	"quarantine.inAnHour":      "za hodinu",
	"quarantine.inHours#one":   "za %d hodinu",
	"quarantine.inHours#few":   "za %d hodiny",
	"quarantine.inHours#other": "za %d hodin",
	"quota.userStorage":        "Místo pro tvoje soubory došlo: smaž staré přílohy nebo poznámky",
	"quota.communityStorage":   "Místo pro média v této komunitě došlo",
	"quota.postRate":           "Příliš mnoho příspěvků za hodinu — zkus to později",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Zpětnou vazbu můžeš poslat jednou denně. Další za méně než hodinu",
	"feedback.nextInHours":     "Zpětnou vazbu můžeš poslat jednou denně. Další za %d h",
	"feedback.replyPushTitle":  "Odpověď na tvou zpětnou vazbu",
	"announcements.dailyLimit": "Denní limit vyčerpán: nejvýše %d hromadných a %d osobních sdělení za 24 hodin",
	"releases.pushTitle":       "Vyšla verze %s",

	// --- pushes ---
	"push.mentioned":  "Někdo tě zmínil ve zprávě",
	"push.newMessage": "Nová zpráva",

	// --- admin ---
	"dashboard.down":          "neodpovídá",
	"dashboard.objectStorage": "objektové úložiště",
	"dashboard.inDatabase":    "v databázi",
	"rating.csv.date":         "Datum",
	"rating.csv.project":      "Projekt",
	"rating.csv.task":         "Úkol",
	"rating.csv.income":       "Příjem",
	"rating.csv.expense":      "Výdaj",
	"rating.csv.profit":       "Zisk",
	"rating.csv.author":       "Autor",
	"rating.csv.comment":      "Komentář",
}

func init() { catalogs["cs"] = cs }
