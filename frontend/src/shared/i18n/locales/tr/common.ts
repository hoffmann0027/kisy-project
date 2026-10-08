import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Silinmiş hesap",
  "common.today": "Bugün",
  "common.yesterday": "Dün",
  "common.justNow": "az önce",
  "common.minutesAgo": "{count} dk önce",
  "common.hoursAgo": "{count} sa önce",
  "common.rateLimited": "Çok fazla istek. Biraz sonra tekrar deneyin",
  "common.rateLimitedSeconds": "Çok fazla istek. {count} sn sonra tekrar deneyin",
  "common.rateLimitedMinutes": "Çok fazla istek. {count} dk sonra tekrar deneyin",

  "common.loading": "Yükleniyor",
  "common.modal.close": "Kapat",
  "common.code.copy": "Kopyala",

  "common.verified.account": "Doğrulanmış hesap",
  "common.verified.community": "Doğrulanmış topluluk",

  "common.media.resetZoom": "Yakınlaştırmayı sıfırla (0)",
  "common.media.download": "İndir",
  "common.media.close": "Kapat (Esc)",
  "common.media.previous": "Önceki",
  "common.media.next": "Sonraki",

  "common.emoji.picker": "Emoji seçici",
  "common.emoji.search": "Emoji ara",
  "common.emoji.noResults": "Sonuç bulunamadı",
  "common.emoji.recent": "Son kullanılanlar",
  "common.emoji.smileys": "İfadeler",
  "common.emoji.gestures": "El hareketleri",
  "common.emoji.hearts": "Kalpler",
  "common.emoji.objects": "Nesneler",

  "common.nav.messages": "Mesajlar",
  "common.nav.communities": "Topluluklar",
  "common.nav.rating": "Sıralama",
  "common.nav.feed": "Akış",

  "common.push.channelName": "Mesajlar",
  "common.push.channelDescription": "Yeni mesajlar ve bahsetmeler",

  "common.displayName.taken": "Bu ad alınmış",
  "common.displayName.letters": "Yalnızca harfler ve kelimeler arasında tek boşluk",
  "common.displayName.length": "Ad: 2 ila 40 karakter",

  "common.password.rule": "12–128 karakter, en az bir harf ve bir rakam",
  "common.password.tooShort": "En az {min} karakter",
  "common.password.tooLong": "En fazla {max} karakter",
  "common.password.needLetter": "En az bir harf gerekli",
  "common.password.needDigit": "En az bir rakam gerekli",

  "common.quarantine.inAnHour": "bir saat sonra",
  "common.quarantine.inHours": {
    one: "{count} saat sonra",
    other: "{count} saat sonra",
  },
  // Turkish puts the verb last: the feature names ("Gönderi paylaşımı") carry
  // no verb, this line adds it after the time.
  "common.quarantine.heldBack": "{feature} {when} açılacak",
  "common.quarantine.fileTooLarge": "Yeni hesaplar en fazla {size} boyutunda dosya gönderebilir. Bu sınır {when} kalkacak",

  "common.units.bytes": "{value} B",
  "common.units.kb": "{value} KB",
  "common.units.mb": "{value} MB",
  "common.units.gb": "{value} GB",

  "common.duration.oneDay": "1 gün",
  "common.duration.days": "{count} gün",
  "common.duration.oneHour": "1 saat",
  "common.duration.hours": "{count} saat",
  "common.duration.minutes": "{count} dk",
  "common.duration.seconds": "{count} sn",

  "common.group.group": "Grup",
  "common.group.community": "Topluluk",
  "common.group.openCommunity": "Herkese açık topluluk",
  "common.group.fromRole": "{kind} · {role} ve üstü",

  "common.session.refreshFailed": "Oturum yenilenemedi",
  "common.consent.saveFailed": "Onayınız kaydedilemedi. Tekrar deneyin",

  "common.offline.title": "Sunucuyla bağlantı yok",
  "common.offline.body": "Hesabınızdan çıkış yapılmadı — internet geri gelir gelmez uygulama kendiliğinden bağlanacak.",
  "common.offline.retry": "Tekrar dene",

  "common.e2ee.deviceNotInChat":
    "Bu cihaz henüz şifreli sohbete eklenmedi. Karşı taraf veya diğer cihazlarınızdan biri çevrimiçi olur olmaz bu cihazı ekleyecek — sonra tekrar gönderin.",
  "common.e2ee.peerNoDevices":
    "Karşı taraf henüz şifreleme destekli bir KISY sürümüyle giriş yapmadı — mesaj gönderilmedi.",
  "common.e2ee.peerKeysExhausted":
    "Karşı tarafın şifreleme anahtarları tükendi — mesaj gönderilmedi. KISY uygulamasını açmasını isteyin ve tekrar deneyin.",
  "common.e2ee.unavailable":
    "Bu cihazda şifreleme başlatılamadı — mesaj gönderilmedi. Uygulamayı yeniden başlatıp tekrar deneyin.",
  "common.e2ee.peerUnknown": "Karşı taraf belirlenemedi — mesaj gönderilmedi.",
  "common.e2ee.encryptFailed": "Mesaj şifrelenemedi — gönderilmedi. Tekrar deneyin.",

  "common.forward.nothingToForward": "İletilebilecek mesaj yok",
};
