import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Gelöschtes Konto",
  "common.today": "Heute",
  "common.yesterday": "Gestern",
  "common.justNow": "gerade eben",
  "common.minutesAgo": "vor {count} Min.",
  "common.hoursAgo": "vor {count} Std.",
  "common.rateLimited": "Zu viele Versuche. Versuch es etwas später erneut",
  "common.rateLimitedSeconds": "Zu viele Versuche. Versuch es in {count} Sek. erneut",
  "common.rateLimitedMinutes": "Zu viele Versuche. Versuch es in {count} Min. erneut",

  "common.loading": "Wird geladen",
  "common.modal.close": "Schließen",
  "common.code.copy": "Kopieren",

  "common.verified.account": "Verifiziertes Konto",
  "common.verified.community": "Verifizierte Community",

  "common.media.resetZoom": "Zoom zurücksetzen (0)",
  "common.media.download": "Herunterladen",
  "common.media.close": "Schließen (Esc)",
  "common.media.previous": "Vorheriges",
  "common.media.next": "Nächstes",

  "common.emoji.picker": "Emoji-Auswahl",
  "common.emoji.search": "Emoji suchen",
  "common.emoji.noResults": "Nichts gefunden",
  "common.emoji.recent": "Zuletzt verwendet",
  "common.emoji.smileys": "Smileys",
  "common.emoji.gestures": "Gesten",
  "common.emoji.hearts": "Herzen",
  "common.emoji.objects": "Objekte",

  "common.nav.messages": "Nachrichten",
  "common.nav.communities": "Communitys",
  "common.nav.rating": "Ranking",
  "common.nav.feed": "Feed",

  "common.push.channelName": "Nachrichten",
  "common.push.channelDescription": "Neue Nachrichten und Erwähnungen",

  "common.displayName.taken": "Dieser Name ist bereits vergeben",
  "common.displayName.letters": "Nur Buchstaben und einzelne Leerzeichen zwischen Wörtern",
  "common.displayName.length": "Name: 2 bis 40 Zeichen",

  "common.password.rule": "12–128 Zeichen, mindestens ein Buchstabe und eine Ziffer",
  "common.password.tooShort": "Mindestens {min} Zeichen",
  "common.password.tooLong": "Höchstens {max} Zeichen",
  "common.password.needLetter": "Mindestens ein Buchstabe erforderlich",
  "common.password.needDigit": "Mindestens eine Ziffer erforderlich",

  "common.quarantine.inAnHour": "in einer Stunde",
  "common.quarantine.inHours": {
    one: "in {count} Stunde",
    other: "in {count} Stunden",
  },
  "common.quarantine.heldBack": "{feature} wird {when} freigeschaltet",
  "common.quarantine.fileTooLarge": "Neue Konten können Dateien bis {size} senden. Die Beschränkung endet {when}",

  "common.units.bytes": "{value} B",
  "common.units.kb": "{value} KB",
  "common.units.mb": "{value} MB",
  "common.units.gb": "{value} GB",

  "common.duration.oneDay": "1 Tag",
  "common.duration.days": "{count} Tage",
  "common.duration.oneHour": "1 Stunde",
  "common.duration.hours": "{count} Stunden",
  "common.duration.minutes": "{count} Min.",
  "common.duration.seconds": "{count} Sek.",

  "common.group.group": "Gruppe",
  "common.group.community": "Community",
  "common.group.openCommunity": "Öffentliche Community",
  "common.group.fromRole": "{kind} · ab {role} aufwärts",

  "common.session.refreshFailed": "Sitzung konnte nicht erneuert werden",
  "common.consent.saveFailed": "Zustimmung konnte nicht gespeichert werden. Versuch es erneut",

  "common.offline.title": "Keine Verbindung zum Server",
  "common.offline.body": "Du bleibst angemeldet – die App verbindet sich von selbst, sobald das Netz wieder da ist.",
  "common.offline.retry": "Erneut versuchen",

  "common.e2ee.deviceNotInChat":
    "Dieses Gerät ist dem verschlüsselten Chat noch nicht beigetreten. Die andere Person oder eines deiner anderen Geräte fügt es hinzu, sobald sie online sind – sende dann erneut.",
  "common.e2ee.peerNoDevices":
    "Die andere Person hat KISY noch nie mit Verschlüsselung genutzt – die Nachricht wurde nicht gesendet.",
  "common.e2ee.peerKeysExhausted":
    "Der anderen Person sind die Verschlüsselungsschlüssel ausgegangen – die Nachricht wurde nicht gesendet. Bitte sie, KISY zu öffnen, und versuch es dann erneut.",
  "common.e2ee.unavailable":
    "Die Verschlüsselung ist auf diesem Gerät nicht gestartet – die Nachricht wurde nicht gesendet. Starte die App neu und versuch es erneut.",
  "common.e2ee.peerUnknown": "Empfänger konnte nicht ermittelt werden – die Nachricht wurde nicht gesendet.",
  "common.e2ee.encryptFailed": "Nachricht konnte nicht verschlüsselt werden – sie wurde nicht gesendet. Versuch es erneut.",

  "common.forward.nothingToForward": "Keine dieser Nachrichten kann weitergeleitet werden",
};
