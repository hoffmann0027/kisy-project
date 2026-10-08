import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Verwijderd account",
  "common.today": "Vandaag",
  "common.yesterday": "Gisteren",
  "common.justNow": "zojuist",
  "common.minutesAgo": "{count} min geleden",
  "common.hoursAgo": "{count} u geleden",
  "common.rateLimited": "Te veel pogingen. Probeer het zo opnieuw",
  "common.rateLimitedSeconds": "Te veel pogingen. Probeer het over {count} s opnieuw",
  "common.rateLimitedMinutes": "Te veel pogingen. Probeer het over {count} min opnieuw",

  "common.loading": "Laden",
  "common.modal.close": "Sluiten",
  "common.code.copy": "Kopiëren",

  "common.verified.account": "Geverifieerd account",
  "common.verified.community": "Geverifieerde community",

  "common.media.resetZoom": "Zoom herstellen (0)",
  "common.media.download": "Downloaden",
  "common.media.close": "Sluiten (Esc)",
  "common.media.previous": "Vorige",
  "common.media.next": "Volgende",

  "common.emoji.picker": "Emoji kiezen",
  "common.emoji.search": "Emoji zoeken",
  "common.emoji.noResults": "Niets gevonden",
  "common.emoji.recent": "Recent",
  "common.emoji.smileys": "Smileys",
  "common.emoji.gestures": "Gebaren",
  "common.emoji.hearts": "Hartjes",
  "common.emoji.objects": "Objecten",

  "common.nav.messages": "Berichten",
  "common.nav.communities": "Community's",
  "common.nav.rating": "Ranglijst",
  "common.nav.feed": "Feed",

  "common.push.channelName": "Berichten",
  "common.push.channelDescription": "Nieuwe berichten en vermeldingen",

  "common.displayName.taken": "Deze naam is al bezet",
  "common.displayName.letters": "Alleen letters en enkele spaties tussen woorden",
  "common.displayName.length": "Naam: 2 tot 40 tekens",

  "common.password.rule": "12–128 tekens, minstens één letter en één cijfer",
  "common.password.tooShort": "Minstens {min} tekens",
  "common.password.tooLong": "Maximaal {max} tekens",
  "common.password.needLetter": "Minstens één letter nodig",
  "common.password.needDigit": "Minstens één cijfer nodig",

  "common.quarantine.inAnHour": "over een uur",
  "common.quarantine.inHours": {
    one: "over {count} uur",
    other: "over {count} uur",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge": "Een nieuw account kan bestanden tot {size} versturen. Deze beperking vervalt {when}",

  "common.units.bytes": "{value} B",
  "common.units.kb": "{value} KB",
  "common.units.mb": "{value} MB",
  "common.units.gb": "{value} GB",

  "common.duration.oneDay": "1 dag",
  "common.duration.days": "{count} dagen",
  "common.duration.oneHour": "1 uur",
  "common.duration.hours": "{count} uur",
  "common.duration.minutes": "{count} min",
  "common.duration.seconds": "{count} sec",

  "common.group.group": "Groep",
  "common.group.community": "Community",
  "common.group.openCommunity": "Openbare community",
  "common.group.fromRole": "{kind} · {role} en hoger",

  "common.session.refreshFailed": "Kan je sessie niet vernieuwen",
  "common.consent.saveFailed": "Kan je toestemming niet opslaan. Probeer het opnieuw",

  "common.offline.title": "Geen verbinding met de server",
  "common.offline.body": "Je blijft ingelogd — de app maakt vanzelf weer verbinding zodra je online bent.",
  "common.offline.retry": "Opnieuw proberen",

  "common.e2ee.deviceNotInChat":
    "Dit apparaat is nog niet toegevoegd aan de beveiligde chat. Je gesprekspartner of een ander apparaat van jou voegt het toe zodra die online is — verstuur het daarna opnieuw.",
  "common.e2ee.peerNoDevices":
    "Je gesprekspartner heeft KISY nog niet gebruikt met versleuteling — het bericht is niet verstuurd.",
  "common.e2ee.peerKeysExhausted":
    "Je gesprekspartner heeft geen versleutelingssleutels meer — het bericht is niet verstuurd. Vraag diegene om KISY te openen en probeer het opnieuw.",
  "common.e2ee.unavailable":
    "Versleuteling is op dit apparaat niet gestart — het bericht is niet verstuurd. Start de app opnieuw en probeer het nog eens.",
  "common.e2ee.peerUnknown": "Kan de gesprekspartner niet bepalen — het bericht is niet verstuurd.",
  "common.e2ee.encryptFailed": "Kan het bericht niet versleutelen — het is niet verstuurd. Probeer het opnieuw.",

  "common.forward.nothingToForward": "Geen berichten die je kunt doorsturen",
};
