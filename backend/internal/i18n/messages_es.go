package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var es = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "La verificación no está disponible ahora mismo. Inténtalo dentro de un minuto",
	"auth.captchaFailed":      "No se pudo confirmar que no eres un robot. Actualiza la página e inténtalo de nuevo",
	"auth.passwordRule":       "contraseña: de 12 a 128 caracteres, con al menos una letra y un número",

	// --- people and chats ---
	"blocks.self":                 "No puedes bloquear tu propia cuenta",
	"reports.self":                "No puedes denunciar tu propia cuenta",
	"chats.restricted":            "Este usuario ha restringido quién puede escribirle",
	"messages.encryptionRequired": "Los mensajes privados solo se pueden enviar cifrados. Actualiza la app.",

	// --- account ---
	"users.deleteWord":        "ELIMINAR",
	"users.deleteConfirm":     "escribe %s para confirmar la eliminación",
	"users.wrongPassword":     "contraseña incorrecta",
	"users.ownerCannotDelete": "No se puede eliminar la cuenta del propietario: primero transfiere la administración",
	"users.nameTaken":         "Este nombre ya está en uso",
	"users.nameChars":         "Solo letras y un único espacio entre palabras",
	"users.nameLength":        "El nombre debe tener entre 2 y 40 caracteres",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "Solo el CEO puede cambiar el nivel de un grupo",
	"groups.sanctionedDelete":   "La comunidad tiene sanciones de moderación y no se puede eliminar mientras estén vigentes",
	"groups.levelAboveYours":    "No puedes crear un grupo con un nivel de acceso superior al tuyo",
	"groups.membersByOwnerOnly": "Solo el propietario del grupo añade miembros; a las comunidades se une cada uno por su cuenta",
	"posts.fileTooLarge":        "El archivo es demasiado grande",
	"posts.membersOnly":         "Esta comunidad es cerrada: solo sus miembros ven las publicaciones",
	"posts.noRights":            "No tienes permiso para publicar aquí",
	"posts.notCommunity":        "Esto es un grupo, no una comunidad",
	"posts.empty":               "La publicación no puede estar vacía",
	"posts.tooLong":             "La publicación es demasiado larga",
	"boards.defaultTitle":       "Tablero de tareas",
	"boards.columnTodo":         "Pendiente",
	"boards.columnDoing":        "En curso",
	"boards.columnDone":         "Hecho",

	// --- moderation ---
	"moderation.reasonRequired":     "Indica el motivo",
	"moderation.reasonTooLong":      "El motivo no puede superar los 1000 caracteres",
	"moderation.restoreFirst":       "La comunidad está eliminada: restáurala primero",
	"moderation.notDeleted":         "La comunidad no está eliminada",
	"moderation.sanctionPermanent":  "Esta sanción no se puede retirar",
	"moderation.restoreExpired":     "El plazo de restauración ha vencido",
	"moderation.warn.community":     "La comunidad «%s» ha recibido una advertencia (%d de %d): %s",
	"moderation.warn.group":         "El grupo «%s» ha recibido una advertencia (%d de %d): %s",
	"moderation.mute.community":     "La comunidad «%s» está silenciada %s y sus publicaciones no aparecen en Novedades. Motivo: %s",
	"moderation.mute.group":         "El grupo «%s» está silenciado %s y sus publicaciones no aparecen en Novedades. Motivo: %s",
	"moderation.delete.community":   "La comunidad «%s» se ha eliminado: %s",
	"moderation.delete.group":       "El grupo «%s» se ha eliminado: %s",
	"moderation.restored.community": "La comunidad «%s» se ha restaurado",
	"moderation.restored.group":     "El grupo «%s» se ha restaurado",
	"moderation.untilForever":       "indefinidamente",
	"moderation.until":              "hasta el %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Podrás publicar %s",
	"quarantine.communities":   "Podrás crear comunidades %s",
	"quarantine.newChats":      "Por ahora solo puedes iniciar unas pocas conversaciones nuevas al día. Este límite se quitará %s",
	"quarantine.upload":        "Las cuentas nuevas solo pueden enviar archivos más pequeños. Este límite se quitará %s",
	"quarantine.other":         "Esta función estará disponible %s",
	"quarantine.inAnHour":      "en una hora",
	"quarantine.inHours#one":   "en %d hora",
	"quarantine.inHours#other": "en %d horas",
	"quota.userStorage":        "Te has quedado sin espacio para tus archivos: elimina adjuntos o notas antiguos",
	"quota.communityStorage":   "Esta comunidad se ha quedado sin espacio para multimedia",
	"quota.postRate":           "Demasiadas publicaciones en una hora. Inténtalo más tarde",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Puedes enviar una opinión al día. La siguiente, en menos de una hora",
	"feedback.nextInHours":     "Puedes enviar una opinión al día. La siguiente, en %d h",
	"feedback.replyPushTitle":  "Respuesta a tu opinión",
	"announcements.dailyLimit": "Has alcanzado el límite diario: como máximo %d difusiones y %d avisos personales cada 24 horas",
	"releases.pushTitle":       "Ya está disponible la versión %s",

	// --- pushes ---
	"push.mentioned":  "Te mencionaron en un mensaje",
	"push.newMessage": "Nuevo mensaje",

	// --- admin ---
	"dashboard.down":          "sin respuesta",
	"dashboard.objectStorage": "almacenamiento de objetos",
	"dashboard.inDatabase":    "en la base de datos",
	"rating.csv.date":         "Fecha",
	"rating.csv.project":      "Proyecto",
	"rating.csv.task":         "Tarea",
	"rating.csv.income":       "Ingreso",
	"rating.csv.expense":      "Gasto",
	"rating.csv.profit":       "Beneficio",
	"rating.csv.author":       "Autor",
	"rating.csv.comment":      "Comentario",
}

func init() { catalogs["es"] = es }
