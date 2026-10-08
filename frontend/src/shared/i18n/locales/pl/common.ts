import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Usunięte konto",
  "common.today": "Dzisiaj",
  "common.yesterday": "Wczoraj",
  "common.justNow": "przed chwilą",
  "common.minutesAgo": "{count} min temu",
  "common.hoursAgo": "{count} godz. temu",
  "common.rateLimited": "Za dużo prób. Spróbuj trochę później",
  "common.rateLimitedSeconds": "Za dużo prób. Spróbuj ponownie za {count} s",
  "common.rateLimitedMinutes": "Za dużo prób. Spróbuj ponownie za {count} min",

  "common.loading": "Ładowanie",
  "common.modal.close": "Zamknij",
  "common.code.copy": "Kopiuj",

  "common.verified.account": "Zweryfikowane konto",
  "common.verified.community": "Zweryfikowana społeczność",

  "common.media.resetZoom": "Przywróć skalę (0)",
  "common.media.download": "Pobierz",
  "common.media.close": "Zamknij (Esc)",
  "common.media.previous": "Poprzednie",
  "common.media.next": "Następne",

  "common.emoji.picker": "Wybór emoji",
  "common.emoji.search": "Szukaj emoji",
  "common.emoji.noResults": "Nic nie znaleziono",
  "common.emoji.recent": "Ostatnie",
  "common.emoji.smileys": "Buźki",
  "common.emoji.gestures": "Gesty",
  "common.emoji.hearts": "Serca",
  "common.emoji.objects": "Przedmioty",

  "common.nav.messages": "Wiadomości",
  "common.nav.communities": "Społeczności",
  "common.nav.rating": "Ranking",
  "common.nav.feed": "Aktualności",

  "common.push.channelName": "Wiadomości",
  "common.push.channelDescription": "Nowe wiadomości i wzmianki",

  "common.displayName.taken": "To imię jest już zajęte",
  "common.displayName.letters": "Tylko litery i pojedyncze spacje między słowami",
  "common.displayName.length": "Imię: od 2 do 40 znaków",

  "common.password.rule": "12–128 znaków, co najmniej jedna litera i jedna cyfra",
  "common.password.tooShort": "Co najmniej {min} znaków",
  "common.password.tooLong": "Nie więcej niż {max} znaków",
  "common.password.needLetter": "Dodaj co najmniej jedną literę",
  "common.password.needDigit": "Dodaj co najmniej jedną cyfrę",

  "common.quarantine.inAnHour": "za godzinę",
  "common.quarantine.inHours": {
    one: "za {count} godzinę",
    few: "za {count} godziny",
    many: "za {count} godzin",
    other: "za {count} godziny",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge": "Nowe konto może wysyłać pliki do {size}. Limit zniknie {when}",

  "common.units.bytes": "{value} B",
  "common.units.kb": "{value} KB",
  "common.units.mb": "{value} MB",
  "common.units.gb": "{value} GB",

  "common.duration.oneDay": "1 dzień",
  "common.duration.days": "{count} dni",
  "common.duration.oneHour": "1 godzina",
  "common.duration.hours": "{count} godz.",
  "common.duration.minutes": "{count} min",
  "common.duration.seconds": "{count} s",

  "common.group.group": "Grupa",
  "common.group.community": "Społeczność",
  "common.group.openCommunity": "Otwarta społeczność",
  "common.group.fromRole": "{kind} · od {role} wzwyż",

  "common.session.refreshFailed": "Nie udało się odświeżyć sesji",
  "common.consent.saveFailed": "Nie udało się zapisać zgody. Spróbuj ponownie",

  "common.offline.title": "Brak połączenia z serwerem",
  "common.offline.body": "Nie wylogowujemy cię — aplikacja połączy się sama, gdy tylko pojawi się sieć.",
  "common.offline.retry": "Ponów",

  "common.e2ee.deviceNotInChat":
    "To urządzenie nie dołączyło jeszcze do szyfrowanego czatu. Doda je rozmówca lub inne twoje urządzenie, gdy tylko będzie online — wtedy wyślij ponownie.",
  "common.e2ee.peerNoDevices": "Rozmówca nie ma jeszcze KISY z obsługą szyfrowania — wiadomość nie została wysłana.",
  "common.e2ee.peerKeysExhausted":
    "Rozmówcy skończyły się klucze szyfrowania — wiadomość nie została wysłana. Poproś rozmówcę o otwarcie KISY i spróbuj ponownie.",
  "common.e2ee.unavailable":
    "Szyfrowanie nie uruchomiło się na tym urządzeniu — wiadomość nie została wysłana. Uruchom aplikację ponownie i spróbuj jeszcze raz.",
  "common.e2ee.peerUnknown": "Nie udało się ustalić rozmówcy — wiadomość nie została wysłana.",
  "common.e2ee.encryptFailed": "Nie udało się zaszyfrować wiadomości — nie została wysłana. Spróbuj ponownie.",

  "common.forward.nothingToForward": "Brak wiadomości, które można przekazać dalej",
};
