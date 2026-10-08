import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Deleted account",
  "common.today": "Today",
  "common.yesterday": "Yesterday",
  "common.justNow": "just now",
  "common.minutesAgo": "{count} min ago",
  "common.hoursAgo": "{count} h ago",
  "common.rateLimited": "Too many attempts. Try again a little later",
  "common.rateLimitedSeconds": "Too many attempts. Try again in {count} s",
  "common.rateLimitedMinutes": "Too many attempts. Try again in {count} min",

  "common.loading": "Loading",
  "common.modal.close": "Close",
  "common.code.copy": "Copy",

  "common.verified.account": "Verified account",
  "common.verified.community": "Verified community",

  "common.media.resetZoom": "Reset zoom (0)",
  "common.media.download": "Download",
  "common.media.close": "Close (Esc)",
  "common.media.previous": "Previous",
  "common.media.next": "Next",

  "common.emoji.picker": "Emoji picker",
  "common.emoji.search": "Search emoji",
  "common.emoji.noResults": "No emoji found",
  "common.emoji.recent": "Recent",
  "common.emoji.smileys": "Smileys",
  "common.emoji.gestures": "Gestures",
  "common.emoji.hearts": "Hearts",
  "common.emoji.objects": "Objects",

  "common.nav.messages": "Messages",
  "common.nav.communities": "Communities",
  "common.nav.rating": "Rankings",
  "common.nav.feed": "Feed",

  "common.push.channelName": "Messages",
  "common.push.channelDescription": "New messages and mentions",

  "common.displayName.taken": "This name is taken",
  "common.displayName.letters": "Letters only, with single spaces between words",
  "common.displayName.length": "Name must be 2 to 40 characters",

  "common.password.rule": "12–128 characters, with at least one letter and one digit",
  "common.password.tooShort": "At least {min} characters",
  "common.password.tooLong": "No more than {max} characters",
  "common.password.needLetter": "Add at least one letter",
  "common.password.needDigit": "Add at least one digit",

  "common.quarantine.inAnHour": "in an hour",
  "common.quarantine.inHours": {
    one: "in {count} hour",
    other: "in {count} hours",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge": "New accounts can send files up to {size}. This limit lifts {when}",

  "common.units.bytes": "{value} B",
  "common.units.kb": "{value} KB",
  "common.units.mb": "{value} MB",
  "common.units.gb": "{value} GB",

  "common.duration.oneDay": "1 day",
  "common.duration.days": "{count} days",
  "common.duration.oneHour": "1 hour",
  "common.duration.hours": "{count} hours",
  "common.duration.minutes": "{count} min",
  "common.duration.seconds": "{count} sec",

  "common.group.group": "Group",
  "common.group.community": "Community",
  "common.group.openCommunity": "Public community",
  "common.group.fromRole": "{kind} · {role} and above",

  "common.session.refreshFailed": "Couldn't refresh your session",
  "common.consent.saveFailed": "Couldn't save your consent. Please try again",

  "common.offline.title": "Can't reach the server",
  "common.offline.body": "You're still signed in — the app will reconnect on its own as soon as you're back online.",
  "common.offline.retry": "Retry",

  "common.e2ee.deviceNotInChat":
    "This device hasn't joined the encrypted chat yet. The other person or another of your devices will add it as soon as they're online — then send again.",
  "common.e2ee.peerNoDevices":
    "The other person hasn't used an encryption-enabled version of KISY yet — the message wasn't sent.",
  "common.e2ee.peerKeysExhausted":
    "The other person has run out of encryption keys — the message wasn't sent. Ask them to open KISY, then try again.",
  "common.e2ee.unavailable":
    "Encryption didn't start on this device — the message wasn't sent. Restart the app and try again.",
  "common.e2ee.peerUnknown": "Couldn't identify the recipient — the message wasn't sent.",
  "common.e2ee.encryptFailed": "Couldn't encrypt the message — it wasn't sent. Please try again.",

  "common.forward.nothingToForward": "None of these messages can be forwarded",
};
