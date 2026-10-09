package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var fr = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "La vérification est indisponible, réessayez dans une minute",
	"auth.captchaFailed":      "Impossible de confirmer que vous n'êtes pas un robot. Actualisez la page et réessayez",
	"auth.passwordRule":       "mot de passe : 12 à 128 caractères, dont au moins une lettre et un chiffre",

	// --- people and chats ---
	"blocks.self":                 "Vous ne pouvez pas vous bloquer vous-même",
	"reports.self":                "Vous ne pouvez pas vous signaler vous-même",
	"chats.restricted":            "Cet utilisateur a restreint qui peut lui écrire",
	"messages.encryptionRequired": "Les messages privés ne peuvent être envoyés que chiffrés. Mettez à jour l'application.",

	// --- account ---
	"users.deleteWord":        "SUPPRIMER",
	"users.deleteConfirm":     "saisissez %s pour confirmer la suppression",
	"users.wrongPassword":     "mot de passe incorrect",
	"users.ownerCannotDelete": "Le compte du propriétaire ne peut pas être supprimé : transférez d'abord la gestion",
	"users.nameTaken":         "Ce nom est déjà pris",
	"users.nameChars":         "Uniquement des lettres, avec un seul espace entre les mots",
	"users.nameLength":        "Nom : de 2 à 40 caractères",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "Seul le CEO peut modifier le niveau d'un groupe",
	"groups.sanctionedDelete":   "Cette communauté fait l'objet de sanctions de modération — tant qu'elles s'appliquent, elle ne peut pas être supprimée",
	"groups.levelAboveYours":    "Impossible de créer un groupe avec un niveau d'accès supérieur au vôtre",
	"groups.membersByOwnerOnly": "Seul le propriétaire ajoute des membres à un groupe ; une communauté, on la rejoint soi-même",
	"groups.banned":             "Vous avez été banni de cette communauté",
	"groups.founderStays":       "Le fondateur ne peut pas partir : la communauté peut seulement être supprimée",
	"groups.cannotDiscipline":   "Seul quelqu'un ayant un rôle supérieur dans le groupe peut exclure ou bannir",
	"posts.fileTooLarge":        "Le fichier est trop volumineux",
	"posts.membersOnly":         "Communauté privée : seuls les membres voient les publications",
	"posts.noRights":            "Vous n'avez pas le droit de publier ici",
	"posts.notCommunity":        "Ceci est un groupe, pas une communauté",
	"posts.empty":               "Une publication ne peut pas être vide",
	"posts.tooLong":             "La publication est trop longue",
	"boards.defaultTitle":       "Tableau des tâches",
	"boards.columnTodo":         "À faire",
	"boards.columnDoing":        "En cours",
	"boards.columnDone":         "Terminé",

	// --- moderation ---
	"moderation.reasonRequired":     "Indiquez un motif",
	"moderation.reasonTooLong":      "Le motif ne doit pas dépasser 1000 caractères",
	"moderation.restoreFirst":       "La communauté a été supprimée — restaurez-la d'abord",
	"moderation.notDeleted":         "La communauté n'est pas supprimée",
	"moderation.sanctionPermanent":  "Cette sanction ne peut pas être levée",
	"moderation.restoreExpired":     "Le délai de restauration a expiré",
	"moderation.warn.community":     "La communauté « %s » a reçu un avertissement (%d sur %d) : %s",
	"moderation.warn.group":         "Le groupe « %s » a reçu un avertissement (%d sur %d) : %s",
	"moderation.mute.community":     "La communauté « %s » est mise en sourdine %s — ses publications n'apparaissent pas dans le fil d'actualité : %s",
	"moderation.mute.group":         "Le groupe « %s » est mis en sourdine %s — ses publications n'apparaissent pas dans le fil d'actualité : %s",
	"moderation.delete.community":   "La communauté « %s » a été supprimée : %s",
	"moderation.delete.group":       "Le groupe « %s » a été supprimé : %s",
	"moderation.restored.community": "La communauté « %s » a été restaurée",
	"moderation.restored.group":     "Le groupe « %s » a été restauré",
	"moderation.untilForever":       "sans limite de durée",
	"moderation.until":              "jusqu'au %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "La publication sera possible %s",
	"quarantine.communities":   "La création de communautés sera possible %s",
	"quarantine.newChats":      "Pour l'instant, vous ne pouvez lancer que quelques nouvelles conversations par jour. Cette limite sera levée %s",
	"quarantine.upload":        "Un nouveau compte ne peut envoyer que des fichiers plus petits. Cette limite sera levée %s",
	"quarantine.other":         "Cette fonction sera disponible %s",
	"quarantine.inAnHour":      "dans une heure",
	"quarantine.inHours#one":   "dans %d heure",
	"quarantine.inHours#other": "dans %d heures",
	"quota.userStorage":        "Vous n'avez plus d'espace pour vos fichiers : supprimez d'anciennes pièces jointes ou notes",
	"quota.communityStorage":   "Cette communauté n'a plus d'espace pour les médias",
	"quota.postRate":           "Trop de publications en une heure — réessayez plus tard",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Vous pouvez laisser un avis une fois par jour. Prochain avis dans moins d'une heure",
	"feedback.nextInHours":     "Vous pouvez laisser un avis une fois par jour. Prochain avis dans %d h",
	"feedback.replyPushTitle":  "Réponse à votre avis",
	"announcements.dailyLimit": "Limite quotidienne atteinte : pas plus de %d diffusions et %d notifications personnelles par 24 heures",
	"releases.pushTitle":       "La version %s est disponible",

	// --- pushes ---
	"push.mentioned":  "Nouvelle mention dans un message",
	"push.newMessage": "Nouveau message",

	// --- admin ---
	"dashboard.down":          "ne répond pas",
	"dashboard.objectStorage": "stockage objet",
	"dashboard.inDatabase":    "dans la base de données",
	"rating.memberCannotSee":  "Cet utilisateur ne voit pas le projet : son niveau est inférieur au niveau d'accès du projet",
	"rating.csv.date":         "Date",
	"rating.csv.project":      "Projet",
	"rating.csv.task":         "Tâche",
	"rating.csv.income":       "Recette",
	"rating.csv.expense":      "Dépense",
	"rating.csv.profit":       "Bénéfice",
	"rating.csv.author":       "Auteur",
	"rating.csv.comment":      "Commentaire",
}

func init() { catalogs["fr"] = fr }
