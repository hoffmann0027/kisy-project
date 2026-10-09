package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var tr = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "Doğrulama şu anda kullanılamıyor, bir dakika sonra tekrar deneyin",
	"auth.captchaFailed":      "Robot olmadığınız doğrulanamadı. Sayfayı yenileyip tekrar deneyin",
	"auth.passwordRule":       "şifre: 12–128 karakter, en az bir harf ve bir rakam",

	// --- people and chats ---
	"blocks.self":                 "Kendinizi engelleyemezsiniz",
	"reports.self":                "Kendinizi şikayet edemezsiniz",
	"chats.restricted":            "Kullanıcı mesajlaşmayı kısıtladı",
	"messages.encryptionRequired": "Özel mesajlar yalnızca şifreli olarak gönderilebilir. Uygulamayı güncelleyin.",

	// --- account ---
	"users.deleteWord":        "SİL",
	"users.deleteConfirm":     "silmeyi onaylamak için %s yazın",
	"users.wrongPassword":     "şifre yanlış",
	"users.ownerCannotDelete": "Sahip hesabı silinemez: önce yönetimi devredin",
	"users.nameTaken":         "Bu ad alınmış",
	"users.nameChars":         "Yalnızca harfler ve kelimeler arasında tek boşluk",
	"users.nameLength":        "Ad: 2 ila 40 karakter",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "Grubun seviyesini yalnızca CEO değiştirebilir",
	"groups.sanctionedDelete":   "Topluluk moderasyon yaptırımı altında — yaptırımlar sürdükçe silinemez",
	"groups.levelAboveYours":    "Kendi seviyenizden yüksek erişim seviyesiyle grup oluşturamazsınız",
	"groups.membersByOwnerOnly": "Gruba üyeleri yalnızca grup sahibi ekler; topluluklara ise kişiler kendileri katılır",
	"groups.banned":             "Bu toplulukta engellendiniz",
	"groups.founderStays":       "Kurucu ayrılamaz — topluluk yalnızca silinebilir",
	"groups.cannotDiscipline":   "Yalnızca grupta daha yüksek role sahip biri çıkarabilir veya engelleyebilir",
	"posts.fileTooLarge":        "Dosya çok büyük",
	"posts.membersOnly":         "Topluluk kapalı: gönderileri yalnızca üyeler görebilir",
	"posts.noRights":            "Burada paylaşım yapma yetkiniz yok",
	"posts.notCommunity":        "Bu bir topluluk değil, grup",
	"posts.empty":               "Gönderi boş olamaz",
	"posts.tooLong":             "Gönderi çok uzun",
	"boards.defaultTitle":       "Görev panosu",
	"boards.columnTodo":         "Yapılacaklar",
	"boards.columnDoing":        "Yapılıyor",
	"boards.columnDone":         "Tamamlandı",

	// --- moderation ---
	"moderation.reasonRequired":     "Bir neden belirtin",
	"moderation.reasonTooLong":      "Neden en fazla 1000 karakter olabilir",
	"moderation.restoreFirst":       "Topluluk silinmiş — önce geri yükleyin",
	"moderation.notDeleted":         "Topluluk silinmemiş",
	"moderation.sanctionPermanent":  "Bu yaptırım kaldırılamaz",
	"moderation.restoreExpired":     "Geri yükleme süresi doldu",
	"moderation.warn.community":     "“%s” topluluğuna uyarı verildi (%d/%d): %s",
	"moderation.warn.group":         "“%s” grubuna uyarı verildi (%d/%d): %s",
	"moderation.mute.community":     "“%s” topluluğu %s susturuldu — gönderileri akışta gösterilmiyor: %s",
	"moderation.mute.group":         "“%s” grubu %s susturuldu — gönderileri akışta gösterilmiyor: %s",
	"moderation.delete.community":   "“%s” topluluğu silindi: %s",
	"moderation.delete.group":       "“%s” grubu silindi: %s",
	"moderation.restored.community": "“%s” topluluğu geri yüklendi",
	"moderation.restored.group":     "“%s” grubu geri yüklendi",
	"moderation.untilForever":       "süresiz olarak",
	"moderation.until":              "%s (UTC) tarihine kadar",

	// --- limits on new accounts and storage ---
	// Turkish puts the verb last, after the time ("18 saat sonra açılacak").
	"quarantine.posts":         "Gönderi paylaşımı %s açılacak",
	"quarantine.communities":   "Topluluk oluşturma %s açılacak",
	"quarantine.newChats":      "Şimdilik günde yalnızca birkaç yeni sohbet başlatabilirsiniz. Bu sınır %s kalkacak",
	"quarantine.upload":        "Yeni hesaplar yalnızca daha küçük dosyalar gönderebilir. Bu sınır %s kalkacak",
	"quarantine.other":         "Bu özellik %s açılacak",
	"quarantine.inAnHour":      "bir saat sonra",
	"quarantine.inHours#one":   "%d saat sonra",
	"quarantine.inHours#other": "%d saat sonra",
	"quota.userStorage":        "Dosyalarınız için ayrılan alan doldu: eski ekleri veya notları silin",
	"quota.communityStorage":   "Bu topluluğun medya alanı doldu",
	"quota.postRate":           "Son bir saatte çok fazla gönderi paylaşıldı — daha sonra tekrar deneyin",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "Günde bir kez geri bildirim bırakabilirsiniz. Sonraki: bir saatten kısa süre sonra",
	"feedback.nextInHours":     "Günde bir kez geri bildirim bırakabilirsiniz. Sonraki: %d sa sonra",
	"feedback.replyPushTitle":  "Geri bildiriminize yanıt",
	"announcements.dailyLimit": "Günlük sınıra ulaşıldı: 24 saatte en fazla %d toplu duyuru ve %d kişisel bildirim",
	"releases.pushTitle":       "%s sürümü yayınlandı",

	// --- pushes ---
	"push.mentioned":  "Bir mesajda sizden bahsedildi",
	"push.newMessage": "Yeni mesaj",

	// --- admin ---
	"dashboard.down":          "yanıt vermiyor",
	"dashboard.objectStorage": "nesne depolama",
	"dashboard.inDatabase":    "veritabanında",
	"rating.memberCannotSee":  "Bu kullanıcı projeyi göremiyor: seviyesi projenin erişim seviyesinin altında",
	"rating.csv.date":         "Tarih",
	"rating.csv.project":      "Proje",
	"rating.csv.task":         "Görev",
	"rating.csv.income":       "Gelir",
	"rating.csv.expense":      "Gider",
	"rating.csv.profit":       "Kâr",
	"rating.csv.author":       "Yazar",
	"rating.csv.comment":      "Yorum",
}

func init() { catalogs["tr"] = tr }
