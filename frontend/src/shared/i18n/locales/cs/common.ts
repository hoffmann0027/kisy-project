import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Smazaný účet",
  "common.today": "Dnes",
  "common.yesterday": "Včera",
  "common.justNow": "právě teď",
  "common.minutesAgo": "před {count} min",
  "common.hoursAgo": "před {count} h",
  "common.rateLimited": "Příliš mnoho pokusů. Zkus to o něco později",
  "common.rateLimitedSeconds": "Příliš mnoho pokusů. Zkus to znovu za {count} s",
  "common.rateLimitedMinutes": "Příliš mnoho pokusů. Zkus to znovu za {count} min",

  "common.loading": "Načítání",
  "common.modal.close": "Zavřít",
  "common.code.copy": "Kopírovat",

  "common.verified.account": "Ověřený účet",
  "common.verified.community": "Ověřená komunita",

  "common.media.resetZoom": "Obnovit přiblížení (0)",
  "common.media.download": "Stáhnout",
  "common.media.close": "Zavřít (Esc)",
  "common.media.previous": "Předchozí",
  "common.media.next": "Další",

  "common.emoji.picker": "Výběr emoji",
  "common.emoji.search": "Hledat emoji",
  "common.emoji.noResults": "Nic nenalezeno",
  "common.emoji.recent": "Nedávné",
  "common.emoji.smileys": "Smajlíci",
  "common.emoji.gestures": "Gesta",
  "common.emoji.hearts": "Srdce",
  "common.emoji.objects": "Předměty",

  "common.nav.messages": "Zprávy",
  "common.nav.communities": "Komunity",
  "common.nav.rating": "Žebříček",
  "common.nav.feed": "Feed",

  "common.push.channelName": "Zprávy",
  "common.push.channelDescription": "Nové zprávy a zmínky",

  "common.displayName.taken": "Toto jméno je obsazené",
  "common.displayName.letters": "Jen písmena, mezi slovy vždy jedna mezera",
  "common.displayName.length": "Jméno: 2 až 40 znaků",

  "common.password.rule": "12–128 znaků, alespoň jedno písmeno a jedna číslice",
  "common.password.tooShort": "Alespoň {min} znaků",
  "common.password.tooLong": "Nejvýše {max} znaků",
  "common.password.needLetter": "Přidej alespoň jedno písmeno",
  "common.password.needDigit": "Přidej alespoň jednu číslici",

  "common.quarantine.inAnHour": "za hodinu",
  "common.quarantine.inHours": {
    one: "za {count} hodinu",
    few: "za {count} hodiny",
    many: "za {count} hodin",
    other: "za {count} hodin",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge": "Nový účet může posílat soubory do velikosti {size}. Omezení skončí {when}",

  "common.units.bytes": "{value} B",
  "common.units.kb": "{value} kB",
  "common.units.mb": "{value} MB",
  "common.units.gb": "{value} GB",

  "common.duration.oneDay": "1 den",
  "common.duration.days": "{count} d",
  "common.duration.oneHour": "1 hodina",
  "common.duration.hours": "{count} h",
  "common.duration.minutes": "{count} min",
  "common.duration.seconds": "{count} s",

  "common.group.group": "Skupina",
  "common.group.community": "Komunita",
  "common.group.openCommunity": "Veřejná komunita",
  "common.group.fromRole": "{kind} · {role} a výše",

  "common.session.refreshFailed": "Relaci se nepodařilo obnovit",
  "common.consent.saveFailed": "Souhlas se nepodařilo uložit. Zkus to znovu",

  "common.offline.title": "Nelze se spojit se serverem",
  "common.offline.body": "Z účtu tě to neodhlásí — aplikace se připojí sama, jakmile bude k dispozici síť.",
  "common.offline.retry": "Zkusit znovu",

  "common.e2ee.deviceNotInChat":
    "Toto zařízení ještě není připojené k šifrovanému chatu. Připojí ho protějšek nebo jiné tvoje zařízení, jakmile bude online — pak zprávu pošli znovu.",
  "common.e2ee.peerNoDevices": "Protějšek se ještě nepřihlásil do verze KISY s podporou šifrování — zpráva nebyla odeslána.",
  "common.e2ee.peerKeysExhausted":
    "Protějšku došly šifrovací klíče — zpráva nebyla odeslána. Požádej ho, ať otevře KISY, a zkus to znovu.",
  "common.e2ee.unavailable":
    "Šifrování se na tomto zařízení nespustilo — zpráva nebyla odeslána. Restartuj aplikaci a zkus to znovu.",
  "common.e2ee.peerUnknown": "Nepodařilo se určit příjemce — zpráva nebyla odeslána.",
  "common.e2ee.encryptFailed": "Zprávu se nepodařilo zašifrovat — nebyla odeslána. Zkus to znovu.",

  "common.forward.nothingToForward": "Žádnou z těchto zpráv nelze přeposlat",
};
