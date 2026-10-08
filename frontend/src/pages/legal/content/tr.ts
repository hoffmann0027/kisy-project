// content/ru.ts dosyasının Türkçe çevirisi. Hukuken bağlayıcı olan Rusça metindir; bu çeviri bilgi amaçlıdır.
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-07",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "Kısaca",
      body: [
        "KISY bir mesajlaşma uygulamasıdır. Çalışabilmesi için hesabınızı, yazışmalarınızı ve gönderdiğiniz dosyaları saklarız. Bu verileri satmayız, reklam göstermeyiz ve aşağıda açıklanan durumlar dışında üçüncü kişilere aktarmayız.",
        "Bire bir özel yazışmalar cihazlarda şifrelenir: sunucu bunları kendisinin okuyamayacağı bir biçimde saklar. Gruplardaki mesajlar ve topluluklardaki gönderiler açık biçimde saklanır — bunları sunucu yöneticisi görebilir.",
        "Hesabınızı istediğiniz zaman silebilirsiniz: Profil → “Hesabı sil” yoluyla ya da uygulamaya giriş yapmadan hesap silme sayfasından.",
      ],
    },
    {
      title: "Verilerden kim sorumludur",
      body: [
        "Hizmet özel bir proje olarak işletilmektedir. Veri sorumlusu, kaydolduğunuz KISY kurulumunun sahibidir; verilerinizle ilgili sorularınızı da belgenin sonunda belirtilen adres üzerinden o yanıtlar.",
        "Uygulama ücretsizdir ve erişilebilirlik ile verilerin korunmasına ilişkin herhangi bir garanti olmaksızın “olduğu gibi” sunulur (bkz. “Sorumluluk” bölümü).",
      ],
    },
    {
      title: "Hangi verileri topluyoruz",
      body: [
        "— Hesap: kullanıcı adı, görünen ad, şifrenin özeti (hash; şifrenin kendisi saklanmaz), kayıt tarihi, erişim seviyesi, yüklediyseniz profil fotoğrafı.",
        "— İçerik: mesajlar, dosyalar, sesli mesajlar, notlar, topluluklardaki gönderiler, tepkiler, takvim ve görevler — uygulamada oluşturduğunuz her şey.",
        "— Teknik veriler: giriş zamanları, IP adresinin geri döndürülemez parmak izi (hash), cihaz ve tarayıcı adı, anlık bildirim tanımlayıcıları, yöneticilerin işlem kaydı.",
        "— Aramalar: aramanın yapıldığı bilgisi, zamanı ve süresi. Aramanın içeriği kaydedilmez.",
        "— Onay: bu politikayı ve topluluk kurallarını ne zaman ve hangi sürümlerini kabul ettiğiniz ile o andaki IP adresinin geri döndürülemez parmak izi. Bu, onayın kanıtıdır; bu nedenle güvenlik kaydı gibi hesap silindikten sonra da saklanır.",
        "Telefon numaranızı, adresinizi, ödeme bilgilerinizi ve kesin konumunuzu istemeyiz ve saklamayız.",
      ],
    },
    {
      title: "Bu veriler neden gerekli",
      body: [
        "— Mesajları iletmek ve yazışmaları cihazlarınızda göstermek için.",
        "— Hizmeti korumak için: spam'i ve şifre deneme saldırılarını sınırlamak, kötüye kullanımı tespit etmek (IP adresinin kendisi değil, hash'i bu yüzden gereklidir).",
        "— Bildirimleri açtıysanız size bildirim göndermek için.",
        "— Bizim için geçerli olduğu durumlarda yasal yükümlülükleri yerine getirmek için.",
        "Verilerinizi reklam, profilleme ve model eğitimi için kullanmayız.",
      ],
    },
    {
      title: "Veriler kimlerin erişimine açılır",
      body: [
        "— Diğer kullanıcılara — yalnızca yazdığınız kişiye ve yazdığınız grupların ve toplulukların üyelerine.",
        "— Hizmetin onlarsız çalışamayacağı altyapı sağlayıcılarına: uygulama ve veritabanı barındırma, dosya depolama, anlık bildirim iletim hizmeti (Google Firebase), bot koruması (Cloudflare Turnstile). Bu sağlayıcılar verileri bizim talimatımızla işler ve kendi amaçları için kullanmaz.",
        "— Sunucu yöneticisine — “Yönetici neleri görür” bölümünde açıklanan kapsamda.",
        "Verileri satmayız ve reklam ağlarına aktarmayız.",
      ],
    },
    {
      title: "Yönetici neleri görür",
      body: [
        "Şifrelemenin sınırları hakkında dürüst olalım. Sunucu yöneticisi bire bir özel yazışmaları okuyamaz: bunlar cihazlarda kalan anahtarlarla şifrelenir.",
        "Yöneticinin görebilecekleri: grup sohbetlerindeki mesajlar ve topluluklardaki gönderiler, adlar ve kullanıcı adları, yazışmanın yapıldığı ve zamanı (kimin kiminle ve ne zaman yazıştığı), yüklenen dosyalar, işlem kaydı. Bir özel mesaj şikayet edildiğinde yalnızca şikayetin yapıldığını görürüz — metin şifreli kalır.",
        "Yazışmalarınızın tamamen gizli kalması gerekiyorsa gruplar yerine özel sohbetleri kullanın.",
      ],
    },
    {
      title: "Verileri ne kadar süre saklıyoruz",
      body: [
        "Mesajlar ve dosyalar, siz onları silene veya hesabınızı silene kadar saklanır. Süreli mesajlar, ayarladığınız zamanlayıcıya göre silinir.",
        "İşlem kaydı ve giriş kayıtları bir yıla kadar saklanır — hesapların ele geçirilmesi olaylarını incelemek için gereklidir.",
        "Veritabanı yedekleri şifrelenmiş olarak 30 güne kadar saklanır. Silinen veriler, yedekler eskidikçe onlardan da kaybolur.",
      ],
    },
    {
      title: "Hesap silme",
      body: [
        "Hesabınızı uygulamada silebilirsiniz: Profil → “Hesabı sil”, ardından şifrenizle onay. Aynı işlem, uygulamayı yüklemeden hesap silme sayfasından da yapılabilir.",
        "Silme hemen gerçekleşir ve geri alınamaz. Silinenler: şifre ve tüm oturumlar, şifreleme anahtarları, anlık bildirim token'ları, dosyalarınız ve notlarınız, ayarlar, tepkiler ve oylar, özel mesajlarınızın metinleri — karşı taraftakiler de dahil.",
        "Kalanlar: grup sohbetlerindeki mesajlarınız ve topluluklardaki gönderileriniz (bunlar başkalarının yazışmaları ve herkese açık akıştır) — artık adınız olmadan, “Silinmiş hesap” adıyla; geriye dönük olarak değiştirilemeyen güvenlik kaydı.",
        "Yönettiğiniz gruplar ve topluluklar yönetimde sıradaki kişiye geçer; böyle biri yoksa hesapla birlikte silinir. Kullanıcı adınız başka hiç kimseye verilmez.",
      ],
    },
    {
      title: "Haklarınız",
      body: [
        "Verilerinizin bir kopyasını alabilir, verilerinizi düzeltebilir, hesabınızı silebilir veya verilerinizin işlenmesine itiraz edebilirsiniz — belgenin sonundaki adrese yazın. Makul bir süre içinde, genellikle 30 gün içinde yanıt veririz.",
        "AB'de veya Birleşik Krallık'ta bulunuyorsanız, ülkenizdeki denetim makamına şikayette bulunma hakkınız vardır.",
      ],
    },
    {
      title: "Güvenlik ve sınırları",
      body: [
        "Neler yapıyoruz: şifreler yalnızca hash olarak (Argon2id) saklanır, özel yazışmalar cihazlarda şifrelenir, trafik TLS üzerinden iletilir, yönetici işlemleri kayda geçirilir, yedekler şifrelenir; şifre tahmin saldırılarına ve spam kayıtlarına karşı koruma vardır.",
        "Neleri vaat etmiyoruz. Hiçbir hizmet mutlak olarak korunmuş değildir. Önceden bilmeniz gereken riskler:",
        "— Sunucunun veya sağlayıcı altyapısının ele geçirilmesi, özel yazışmaların içeriği dışında her şeyi açığa çıkarabilir.",
        "— Cihazınıza erişim (hırsızlık, kötü amaçlı yazılım, başkasının eline geçmesi) özel yazışmaları da açığa çıkarır: anahtarlar cihazda saklanır.",
        "— Karşı taraf, ona gönderdiğiniz şeyi kaydedebilir, iletebilir veya fotoğrafını çekebilir. Bunu teknik olarak önlemek mümkün değildir.",
        "— Cihazın kaybolması veya uygulamanın yeniden yüklenmesi, özel yazışma geçmişinin kaybı anlamına gelebilir: şifreleme anahtarları cihazdan çıkmaz ve bunları sizin yerinize kurtaramayız.",
        "— Barındırma arızası, yazılım hatası veya ücretsiz planın sınırlarının dolması, hizmetin erişilemez olmasına ve veri kaybına yol açabilir. Önemli verilerinizi kendiniz yedekleyin.",
        "— Anlık bildirimler Google üzerinden geçer ve cihaza giderken tarafımızdan şifrelenmez; önizlemeyi kapatmadıysanız bildirimde gönderenin adı ve mesajın başı yer alabilir.",
      ],
    },
    {
      title: "Çocuklar",
      body: [
        "Hizmet 13 yaşından küçük çocuklar için tasarlanmamıştır (AB'de 16 yaşından veya ülkenizin belirlediği yaştan küçükler için). Bir çocuğun hizmeti ebeveyn izni olmadan kullandığını öğrenirseniz bize yazın; hesabı sileriz.",
      ],
    },
    {
      title: "Sorumluluk",
      body: [
        "Hizmet “olduğu gibi” ve “mevcut olduğu şekliyle”, hiçbir garanti olmaksızın sunulur: erişilebilirlik, verilerin korunması, belirli bir amaca uygunluk ve hatasızlık konusunda garanti verilmez.",
        "Yürürlükteki hukukun izin verdiği azami ölçüde, hizmet sahibi şunlardan sorumlu değildir: verilerin ve yazışmaların kaybı veya bozulması, hizmetin erişilemez olması, şifreleme anahtarlarının ve geçmişin kurtarılamaması, kullanıcıların oluşturduğu içerik, diğer kullanıcıların ve üçüncü kişilerin eylemleri, ayrıca dolaylı zararlar, yoksun kalınan kâr ve hizmetin kullanılmasının veya kullanılamamasının her türlü sonucu.",
        "Yazışmaların içeriğini kullanıcılar oluşturur. Hizmet sahibi bu içeriği önceden kontrol etmez ve ondan sorumlu değildir; kurallar ihlal edildiğinde hesap kısıtlanabilir veya silinebilir.",
        "Yukarıdaki çekinceler, yasa gereği hiçbir çekinceyle kaldırılamayacak olanları ortadan kaldırmaz: kasıtlı eylemlerden ve ağır ihmalden doğan sorumluluk, yaşama ve sağlığa verilen zarardan doğan sorumluluk, ayrıca ülkenizdeki tüketici hakları ve kişisel veri sorumlusunun yükümlülükleri. Böyle bir sınırlamanın kabul edilemediği durumlarda sınırlama, yasanın izin verdiği asgari ölçüde uygulanır.",
      ],
    },
    {
      title: "Değişiklikler",
      body: [
        "Bu politikayı değiştirebiliriz. Önemli değişiklikler uygulamada gösterilir. Son değişiklik tarihi sayfanın üst kısmında belirtilmiştir.",
      ],
    },
    {
      title: "İletişim",
      body: [
        "Verilerle ilgili sorular, silme talepleri ve şikayetler için: kisyandco@gmail.com. Makul bir süre içinde, genellikle 30 gün içinde yanıt veririz.",
      ],
    },
  ],
  deletion: [
    {
      title: "Hesap uygulamadan nasıl silinir",
      body: [
        "1. KISY uygulamasını açın ve hesabınıza giriş yapın.",
        "2. Profil (sağ alttaki simge) → “Hesabı sil”.",
        "3. Şifrenizi ve SİL kelimesini girin.",
        "4. Bitti: hesap hemen silinir, geri alınamaz.",
      ],
    },
    {
      title: "Neler silinir",
      body: [
        "— Şifre, tüm etkin oturumlar ve cihazlar.",
        "— Şifreleme anahtarları ve anlık bildirim token'ları.",
        "— Dosyalarınız, notlarınız, ayarlarınız, tepkileriniz ve oylarınız.",
        "— Özel mesajlarınızın metinleri, karşı taraftakiler de dahil.",
      ],
    },
    {
      title: "Neler kalır",
      body: [
        "— Grup sohbetlerindeki mesajlarınız ve topluluklardaki gönderileriniz — adınız olmadan, “Silinmiş hesap” adıyla. Bunlar başkalarının yazışmaları ve herkese açık akıştır; bunları başkaları adına silemeyiz.",
        "— Güvenlik kaydı (kimin ne zaman giriş yaptığı, yönetici işlemleri) — bir yıla kadar, yazışma içerikleri olmadan.",
        "— Politikayı ve topluluk kurallarını kabul ettiğinize dair kayıt: ne zaman ve hangi sürümlerini. Bu kayıt olmadan onay verildiği kanıtlanamaz.",
        "— Veritabanının şifrelenmiş yedekleri — 30 güne kadar, ardından silinir.",
      ],
    },
    {
      title: "Giriş yapamıyorsanız",
      body: [
        "Silmek istediğiniz kullanıcı adını belirterek kisyandco@gmail.com adresine yazın. Hesabın size ait olduğunu doğruladıktan sonra hesabı sileriz.",
      ],
    },
  ],
  rules: [
    {
      title: "Kısaca",
      body: [
        "KISY; yazışmalar, ortak gruplar ve topluluklar için bir alandır. Aşağıdaki kurallar, bir şey yazdığınız, yüklediğiniz veya başkalarına gösterdiğiniz her yerde geçerlidir: özel ve grup sohbetlerinde, topluluklarda ve akışlarında, grupların adlarında ve açıklamalarında, profilinizdeki ad ve fotoğrafta.",
        "Temel kural: başkalarına, gerçek hayatta yasa ya da insanlar önünde hesabını vermeniz gereken şeyleri yapmayın. Emin değilseniz paylaşmayın.",
      ],
    },
    {
      title: "Neler yasaktır",
      body: [
        "— Çocukların cinsel istismarı ve reşit olmayanları cinselleştiren her türlü materyal. Burada uyarı yoktur: hesap derhal engellenir ve bilgiler kolluk kuvvetlerine iletilir.",
        "— Müstehcen cinsel içerik ve pornografi. Hizmet 13 yaş ve üzeri kullanıcılar için tasarlanmıştır.",
        "— Tehditler, şiddete çağrı, şiddetin ve terörün övülmesi, aşırılıkçı örgütlere eleman kazandırma.",
        "— Zorbalık ve taciz: hakaret, bir kişiye sistematik saldırılar, başkalarını birine karşı kışkırtma, sizi engelleyen veya durmanızı isteyen kişiye tekrar tekrar mesaj gönderme.",
        "— Milliyet, ırk, din, cinsiyet, cinsel yönelim, engellilik, yaş veya köken temelinde nefreti kışkırtma ve insanları aşağılama.",
        "— Başkalarının kişisel verilerini izinsiz yayımlama: adres, telefon, belgeler, özel hayata ait fotoğraflar, yazışmalar.",
        "— Kendine zarar vermeye ve intihara çağrı, bunlara yönelik talimatlar, yeme bozukluklarının romantikleştirilmesi.",
        "— Spam ve sahte etkileşim: toplu olarak gönderilen aynı mesajlar, istenmeyen reklam, hesap çiftlikleri, tepki ve oyların yapay olarak artırılması.",
        "— Dolandırıcılık ve aldatma: kimlik avı (phishing), para ve şifre koparmaya çalışma, sahte çekilişler, kötü amaçlı dosyalar ve bağlantılar.",
        "— Başka bir kişinin veya kuruluşun kimliğine bürünme; yanıltıcı ad ve profil fotoğrafı da buna dahildir.",
        "— Yasaklı şeylerin satışı ve reklamı: uyuşturucu, silah, sahte belgeler, çalıntı mallar.",
        "— Başkalarının haklarını ihlal etme: başkalarına ait eserleri, fotoğrafları ve materyalleri buna hakkınız olmadan yayımlama.",
        "— Kısıtlamaları aşma: önceki hesabın kısıtlanmasına veya silinmesine yol açan şeye devam etmek için yeni hesap açma.",
      ],
    },
    {
      title: "Gruplar ve topluluklar",
      body: [
        "Grubu veya topluluğu oluşturan kişi ve onun atadığı editörler, içerideki düzenden sorumludur ve gönderileri silebilir. Topluluğun iç kuralları, bu kuralların yasakladığı bir şeye izin veremez.",
        "Kapalı topluluk, kuralların geçerli olmadığı bir yer değildir: kapalılık üyeleri yabancı gözlerden korur, ihlalleri moderasyondan değil.",
      ],
    },
    {
      title: "Özel yazışmalar",
      body: [
        "Özel sohbetler cihazlarda şifrelenir ve biz bunları okuyamayız — şikayet durumunda da: özel sohbetteki bir mesaj için yalnızca şikayet yapıldığını görürüz. Bu yüzden burada asıl koruma sizin elinizdedir: karşı tarafı istediğiniz zaman engelleyebilirsiniz; hesapla ilgili kararlar ise şikayetlerin bütününe bakılarak verilir.",
        "Şifreleme, yasak olanı serbest kılmaz. Karşı taraf ihlali kendisi gösterirse — örneğin bize yaptığı başvuruda bir ekran görüntüsüyle — bu kurallar uyarınca önlem alma hakkımız vardır.",
      ],
    },
    {
      title: "Kendinizi nasıl korur ve nasıl şikayet edersiniz",
      body: [
        "— Engelleme. Kişiyle olan özel sohbetin üst kısmındaki düğme. Engelleme tek taraflı ve sessizdir: kişi bundan haberdar olmaz, size özel mesaj yazamaz ve sizi arayamaz; topluluklardaki gönderileri de akışınızdan kaybolur. Engeli Profil → “Engellenenler” bölümünden kaldırabilirsiniz.",
        "— Bir mesajı (kendi menüsünden), topluluktaki bir gönderiyi, bir kişiyi (özel sohbetin üst kısmından veya grup üyeleri listesinden) ve topluluğun ya da grubun kendisini şikayet edebilirsiniz. Şikayet ettiğiniz kişi bundan haberdar olmaz.",
        "— Yalnızca kendi gördüğünüz şeyleri şikayet edebilirsiniz.",
        "— Topluluktaki bir gönderi beş farklı kişi tarafından şikayet edilirse incelenene kadar akıştan gizlenir. Son birkaç saat içinde oluşturulmuş hesaplardan gelen şikayetler yönetim ekibine ulaşır, ancak bu beşe sayılmaz — aksi hâlde bir gönderi bir avuç yeni hesapla gizlenebilirdi.",
        "— Şikayetleri hizmetin yönetim ekibi inceler. Tehdit ediliyorsanız veya tehlikedeyseniz önce polise başvurun: biz bir acil yardım hizmeti değiliz.",
      ],
    },
    {
      title: "İhlallerin sonuçları",
      body: [
        "İhlalin ağırlığına ve tekrarlanıp tekrarlanmadığına göre:",
        "— topluluktaki gönderinin silinmesi — topluluğun editörleri veya yönetim ekibi tarafından;",
        "— grup veya topluluk için: uyarı, genel akıştan çıkarılma, silinme (geçerli üçüncü uyarı topluluğu siler);",
        "— hesap için: engelleme; bundan sonra hesaba giriş yapılamaz.",
        "Ağır ihlallerde — yaşama yönelik tehditler, çocukların cinsel istismarı, terör — hesap uyarı yapılmadan engellenir ve bilgiler yasada belirlenen usule uygun olarak kolluk kuvvetlerine iletilebilir.",
        "Davetsiz oluşturulan bir hesap, kayıttan sonraki ilk saatlerde kısıtlı modda çalışır: bu bir ceza değil, spama karşı bir korumadır.",
      ],
    },
    {
      title: "Karara katılmıyorsanız",
      body: [
        "kisyandco@gmail.com adresine yazın: kullanıcı adınızı, ne olduğunu ve kararın neden hatalı olduğunu düşündüğünüzü belirtin. Kararı yeniden inceleyip genellikle 30 gün içinde yanıt veririz.",
      ],
    },
    {
      title: "Kurallardaki değişiklikler",
      body: [
        "Kurallar değişebilir. Önemli değişikliklerde uygulama, bir sonraki girişinizde yeni sürümü kabul etmenizi ister — kabul etmeden hizmeti kullanamazsınız. Son değişiklik tarihi sayfanın üst kısmında belirtilmiştir.",
      ],
    },
  ],
};
