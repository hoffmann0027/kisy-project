import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Cuenta eliminada",
  "common.today": "Hoy",
  "common.yesterday": "Ayer",
  "common.justNow": "ahora mismo",
  "common.minutesAgo": "hace {count} min",
  "common.hoursAgo": "hace {count} h",
  "common.rateLimited": "Demasiados intentos. Inténtalo de nuevo en un rato",
  "common.rateLimitedSeconds": "Demasiados intentos. Inténtalo de nuevo en {count} s",
  "common.rateLimitedMinutes": "Demasiados intentos. Inténtalo de nuevo en {count} min",

  "common.loading": "Cargando",
  "common.modal.close": "Cerrar",
  "common.code.copy": "Copiar",

  "common.verified.account": "Cuenta verificada",
  "common.verified.community": "Comunidad verificada",

  "common.media.resetZoom": "Restablecer zoom (0)",
  "common.media.download": "Descargar",
  "common.media.close": "Cerrar (Esc)",
  "common.media.previous": "Anterior",
  "common.media.next": "Siguiente",

  "common.emoji.picker": "Selector de emojis",
  "common.emoji.search": "Buscar emojis",
  "common.emoji.noResults": "No se encontró nada",
  "common.emoji.recent": "Recientes",
  "common.emoji.smileys": "Caritas",
  "common.emoji.gestures": "Gestos",
  "common.emoji.hearts": "Corazones",
  "common.emoji.objects": "Objetos",

  "common.nav.messages": "Mensajes",
  "common.nav.communities": "Comunidades",
  "common.nav.rating": "Ranking",
  "common.nav.feed": "Novedades",

  "common.push.channelName": "Mensajes",
  "common.push.channelDescription": "Mensajes nuevos y menciones",

  "common.displayName.taken": "Este nombre ya está en uso",
  "common.displayName.letters": "Solo letras y un único espacio entre palabras",
  "common.displayName.length": "El nombre debe tener entre 2 y 40 caracteres",

  "common.password.rule": "De 12 a 128 caracteres, con al menos una letra y un número",
  "common.password.tooShort": "Mínimo {min} caracteres",
  "common.password.tooLong": "Máximo {max} caracteres",
  "common.password.needLetter": "Añade al menos una letra",
  "common.password.needDigit": "Añade al menos un número",

  "common.quarantine.inAnHour": "en una hora",
  "common.quarantine.inHours": {
    one: "en {count} hora",
    many: "en {count} horas",
    other: "en {count} horas",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge": "Las cuentas nuevas pueden enviar archivos de hasta {size}. Este límite se quitará {when}",

  "common.units.bytes": "{value} B",
  "common.units.kb": "{value} KB",
  "common.units.mb": "{value} MB",
  "common.units.gb": "{value} GB",

  "common.duration.oneDay": "1 día",
  "common.duration.days": "{count} días",
  "common.duration.oneHour": "1 hora",
  "common.duration.hours": "{count} h",
  "common.duration.minutes": "{count} min",
  "common.duration.seconds": "{count} s",

  "common.group.group": "Grupo",
  "common.group.community": "Comunidad",
  "common.group.openCommunity": "Comunidad abierta",
  "common.group.fromRole": "{kind} · {role} o superior",

  "common.session.refreshFailed": "No se pudo renovar la sesión",
  "common.consent.saveFailed": "No se pudo guardar tu consentimiento. Inténtalo de nuevo",

  "common.offline.title": "Sin conexión con el servidor",
  "common.offline.body": "Sigues con la sesión iniciada: la app se reconectará sola en cuanto vuelva la conexión.",
  "common.offline.retry": "Reintentar",

  "common.e2ee.deviceNotInChat":
    "Este dispositivo aún no se ha unido al chat cifrado. Lo añadirá la otra persona u otro de tus dispositivos en cuanto esté en línea; entonces vuelve a enviarlo.",
  "common.e2ee.peerNoDevices":
    "La otra persona aún no ha entrado en KISY con una versión compatible con el cifrado: el mensaje no se envió.",
  "common.e2ee.peerKeysExhausted":
    "A la otra persona se le acabaron las claves de cifrado: el mensaje no se envió. Pídele que abra KISY y vuelve a intentarlo.",
  "common.e2ee.unavailable":
    "El cifrado no se pudo iniciar en este dispositivo: el mensaje no se envió. Reinicia la app y vuelve a intentarlo.",
  "common.e2ee.peerUnknown": "No se pudo identificar a la otra persona: el mensaje no se envió.",
  "common.e2ee.encryptFailed": "No se pudo cifrar el mensaje y no se envió. Vuelve a intentarlo.",

  "common.forward.nothingToForward": "No hay mensajes que se puedan reenviar",
};
