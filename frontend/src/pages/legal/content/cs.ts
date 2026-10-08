// Český překlad content/ru.ts. Právně závazné je ruské znění; tento překlad slouží pro informaci.
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-07",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "Ve zkratce",
      body: [
        "KISY je messenger. Aby fungoval, ukládáme tvůj účet, tvoje konverzace a soubory, které posíláš. Tato data neprodáváme, nezobrazujeme reklamu a nepředáváme je třetím stranám, s výjimkou případů popsaných níže.",
        "Soukromé konverzace mezi dvěma lidmi jsou šifrované na zařízeních: server je ukládá v podobě, kterou sám nedokáže přečíst. Zprávy ve skupinách a příspěvky v komunitách se ukládají nešifrované — vidí je administrátor serveru.",
        "Účet můžeš kdykoli smazat: profil → „Smazat účet“, nebo na stránce pro smazání účtu bez přihlášení do aplikace.",
      ],
    },
    {
      title: "Kdo za data odpovídá",
      body: [
        "Služba funguje jako soukromý projekt. Správcem údajů je vlastník instalace KISY, na které je zaregistrovaný tvůj účet — ten také odpovídá na dotazy k tvým údajům na adrese uvedené na konci dokumentu.",
        "Aplikace je zdarma a poskytuje se „tak, jak je“, bez záruk dostupnosti a zachování dat (viz oddíl „Odpovědnost“).",
      ],
    },
    {
      title: "Jaké údaje shromažďujeme",
      body: [
        "— Účet: uživatelské jméno, zobrazované jméno, hash hesla (samotné heslo se neukládá), datum registrace, úroveň přístupu, avatar, pokud je nahraný.",
        "— Obsah: zprávy, soubory, hlasové zprávy, poznámky, příspěvky v komunitách, reakce, kalendář a úkoly — vše, co v aplikaci vytvoříš.",
        "— Technické údaje: časy přihlášení, nevratný otisk (hash) IP adresy, název zařízení a prohlížeče, identifikátory push oznámení, protokol akcí administrátorů.",
        "— Hovory: skutečnost, čas a délka hovoru. Obsah hovoru se nenahrává.",
        "— Souhlas: kdy přijmeš tyto zásady a pravidla komunity, v jakém znění, a nevratný otisk IP adresy v tu chvíli. Je to doklad o souhlasu, a proto se uchovává i po smazání účtu — stejně jako bezpečnostní protokol.",
        "Nevyžadujeme ani neukládáme telefonní číslo, adresu, platební údaje ani přesnou polohu.",
      ],
    },
    {
      title: "K čemu to potřebujeme",
      body: [
        "— Abychom mohli doručovat zprávy a zobrazovat konverzace na tvých zařízeních.",
        "— Abychom chránili službu: omezovali spam a hádání hesel a odhalovali zneužití (k tomu slouží hash IP adresy, ne samotná adresa).",
        "— Abychom ti posílali oznámení, pokud je máš zapnutá.",
        "— Abychom plnili požadavky zákona, pokud se na nás vztahují.",
        "Tvoje údaje nepoužíváme k reklamě, profilování ani k trénování modelů.",
      ],
    },
    {
      title: "Komu jsou údaje přístupné",
      body: [
        "— Ostatním uživatelům — přesně tomu, komu píšeš, a členům skupin a komunit, ve kterých píšeš.",
        "— Poskytovatelům infrastruktury, bez kterých služba nefunguje: hosting aplikace a databáze, úložiště souborů, služba pro doručování push oznámení (Google Firebase), ochrana proti botům (Cloudflare Turnstile). Zpracovávají údaje na náš pokyn a pro vlastní účely je nepoužívají.",
        "— Administrátorovi serveru — v rozsahu popsaném v oddílu „Co vidí administrátor“.",
        "Údaje neprodáváme ani je nepředáváme reklamním sítím.",
      ],
    },
    {
      title: "Co vidí administrátor",
      body: [
        "Upřímně o hranicích šifrování. Administrátor serveru nemůže číst soukromé konverzace mezi dvěma lidmi: jsou zašifrované klíči, které zůstávají na zařízeních.",
        "Administrátor může vidět: zprávy ve skupinových chatech a příspěvky v komunitách, jména a uživatelská jména, skutečnost a čas konverzace (kdo s kým a kdy), nahrané soubory, protokol akcí. Při nahlášení soukromé zprávy vidíme jen to, že nahlášení existuje — text zůstává zašifrovaný.",
        "Pokud potřebuješ naprostou důvěrnost komunikace, používej soukromé chaty, ne skupiny.",
      ],
    },
    {
      title: "Jak dlouho údaje uchováváme",
      body: [
        "Zprávy a soubory se uchovávají, dokud je nesmažeš nebo dokud nesmažeš účet. Mizející zprávy se mažou podle časovače, který nastavíš.",
        "Protokol akcí a záznamy o přihlášeních se uchovávají až jeden rok — jsou potřeba k vyšetřování hackerských útoků.",
        "Zálohy databáze se uchovávají v zašifrované podobě až 30 dní. Smazaná data ze záloh mizí, jak zálohy postupně zastarávají.",
      ],
    },
    {
      title: "Smazání účtu",
      body: [
        "Účet můžeš smazat v aplikaci: profil → „Smazat účet“, potvrzení heslem. Totéž je k dispozici na stránce pro smazání účtu, bez instalace aplikace.",
        "Smazání proběhne okamžitě a nelze ho vrátit. Smažou se: heslo a všechny relace, šifrovací klíče, push tokeny, tvoje soubory a poznámky, nastavení, reakce a hlasy v hlasováních, texty tvých soukromých zpráv — a to i u protějšku.",
        "Zůstanou: tvoje zprávy ve skupinových chatech a příspěvky v komunitách (jde o konverzace jiných lidí a veřejný feed) — už bez jména, pod označením „Smazaný účet“; bezpečnostní protokol, který nelze zpětně přepisovat.",
        "Skupiny a komunity pod tvou správou přejdou na dalšího správce v pořadí; pokud takový není, smažou se spolu s účtem. Tvoje uživatelské jméno nikdy nedostane nikdo jiný.",
      ],
    },
    {
      title: "Tvoje práva",
      body: [
        "Můžeš získat kopii svých údajů, opravit je, smazat účet nebo vznést námitku proti zpracování — napiš nám na adresu uvedenou na konci dokumentu. Odpovídáme v přiměřené lhůtě, obvykle do 30 dnů.",
        "Pokud jsi v EU nebo ve Spojeném království, máš právo podat stížnost u dozorového úřadu své země.",
      ],
    },
    {
      title: "Bezpečnost a její meze",
      body: [
        "Co děláme: hesla ukládáme jen jako hashe (Argon2id), soukromé konverzace se šifrují na zařízeních, provoz jde přes TLS, akce administrátorů se zaznamenávají, zálohy se šifrují a máme ochranu proti hádání hesel a spamovým registracím.",
        "Co neslibujeme. Žádná služba není chráněná absolutně. Rizika, o kterých musíš vědět předem:",
        "— Napadení serveru nebo infrastruktury poskytovatele může odhalit vše kromě obsahu soukromých konverzací.",
        "— Přístup k tvému zařízení (krádež, škodlivý program, cizí ruce) odhalí i soukromé konverzace: klíče jsou uložené v zařízení.",
        "— Protějšek si může uložit, přeposlat nebo vyfotit to, co mu pošleš. Technicky tomu nelze zabránit.",
        "— Ztráta zařízení nebo přeinstalace aplikace může znamenat ztrátu historie soukromých konverzací: šifrovací klíče zařízení neopouštějí a obnovit je za tebe nemůžeme.",
        "— Výpadek hostingu, chyba v programu nebo vyčerpání bezplatného tarifu mohou vést k nedostupnosti služby a ke ztrátě dat. Důležité věci si zálohuj samostatně.",
        "— Push oznámení procházejí přes Google a na cestě do zařízení je nešifrujeme; může se v nich objevit jméno odesílatele a začátek zprávy, pokud nemáš vypnutý náhled.",
      ],
    },
    {
      title: "Děti",
      body: [
        "Služba není určena dětem mladším 13 let (v EU mladším 16 let nebo věku stanoveného tvou zemí). Pokud zjistíš, že službu používá dítě bez souhlasu rodičů, napiš nám a účet smažeme.",
      ],
    },
    {
      title: "Odpovědnost",
      body: [
        "Služba se poskytuje „tak, jak je“ a „tak, jak je dostupná“, bez jakýchkoli záruk: dostupnosti, zachování dat, vhodnosti pro konkrétní účel a bezchybnosti.",
        "V maximálním rozsahu, který připouštějí použitelné právní předpisy, vlastník služby neodpovídá za: ztrátu nebo poškození dat a konverzací, nedostupnost služby, nemožnost obnovit šifrovací klíče a historii, obsah vytvořený uživateli, jednání jiných uživatelů a třetích stran, ani za nepřímé škody, ušlý zisk a jakékoli důsledky používání nebo nemožnosti používání služby.",
        "Obsah konverzací vytvářejí uživatelé. Vlastník služby ho předem nekontroluje a neodpovídá za něj; při porušení pravidel může být účet omezen nebo smazán.",
        "Výše uvedené výhrady neruší to, čeho se podle zákona nelze zbavit: odpovědnost za úmyslné jednání a hrubou nedbalost, za újmu na životě a zdraví, ani práva spotřebitelů a povinnosti správce osobních údajů ve tvé zemi. Tam, kde je takové omezení nepřípustné, uplatní se jen v minimálním rozsahu, který zákon připouští.",
      ],
    },
    {
      title: "Změny",
      body: [
        "Tyto zásady můžeme měnit. Podstatné změny se zobrazí v aplikaci. Datum poslední změny je uvedeno v horní části stránky.",
      ],
    },
    {
      title: "Kontakt",
      body: [
        "Dotazy k údajům, žádosti o smazání a stížnosti: kisyandco@gmail.com. Odpovídáme v přiměřené lhůtě, obvykle do 30 dnů.",
      ],
    },
  ],
  deletion: [
    {
      title: "Jak smazat účet v aplikaci",
      body: [
        "1. Otevři KISY a přihlas se ke svému účtu.",
        "2. Profil (ikona vpravo dole) → „Smazat účet“.",
        "3. Zadej heslo a slovo SMAZAT.",
        "4. Hotovo: účet se smaže okamžitě a vrátit to nelze.",
      ],
    },
    {
      title: "Co se smaže",
      body: [
        "— Heslo, všechny aktivní relace a zařízení.",
        "— Šifrovací klíče a push tokeny.",
        "— Tvoje soubory, poznámky, nastavení, reakce a hlasy v hlasováních.",
        "— Texty tvých soukromých zpráv, a to i u protějšku.",
      ],
    },
    {
      title: "Co zůstane",
      body: [
        "— Tvoje zprávy ve skupinových chatech a příspěvky v komunitách — bez jména, pod označením „Smazaný účet“. Jde o konverzace jiných lidí a veřejný feed a mazat je za ostatní nemůžeme.",
        "— Bezpečnostní protokol (kdo a kdy se přihlásil, akce administrátorů) — až jeden rok, bez obsahu konverzací.",
        "— Záznam o přijetí zásad a pravidel komunity: kdy a v jakém znění. Bez něj nelze prokázat, že souhlas byl udělen.",
        "— Zašifrované zálohy databáze — až 30 dní, poté zmizí.",
      ],
    },
    {
      title: "Pokud se nemůžeš přihlásit",
      body: [
        "Napiš na kisyandco@gmail.com a uveď uživatelské jméno, které chceš smazat. Účet smažeme po ověření, že je tvůj.",
      ],
    },
  ],
  rules: [
    {
      title: "Ve zkratce",
      body: [
        "KISY je místo pro konverzace, společné skupiny a komunity. Níže uvedená pravidla platí všude, kde něco píšeš, nahráváš nebo ukazuješ ostatním: v soukromých i skupinových chatech, v komunitách a jejich feedu, v názvech a popisech skupin, ve jménu a avataru profilu.",
        "Hlavní pravidlo: nedělej ostatním nic, za co se v běžném životě odpovídá před zákonem nebo před lidmi. V případě pochybností nepublikuj.",
      ],
    },
    {
      title: "Co je zakázáno",
      body: [
        "— Sexuální zneužívání dětí a jakékoli materiály, které sexualizují nezletilé. Tady žádná varování nejsou: účet se okamžitě zablokuje a informace se předají orgánům činným v trestním řízení.",
        "— Explicitní sexuální obsah a pornografie. Služba je určena uživatelům od 13 let.",
        "— Výhrůžky, výzvy k násilí, oslavování násilí a terorismu, nábor do extremistických organizací.",
        "— Šikana a pronásledování: urážky, soustavné útoky na člověka, navádění ostatních proti němu, opakované zprávy tomu, kdo tě zablokoval nebo tě požádal, ať toho necháš.",
        "— Podněcování k nenávisti a ponižování lidí kvůli národnosti, rase, náboženství, pohlaví, sexuální orientaci, zdravotnímu postižení, věku nebo původu.",
        "— Zveřejňování cizích osobních údajů bez souhlasu: adresy, telefonu, dokladů, fotografií ze soukromého života, korespondence.",
        "— Výzvy k sebepoškozování a sebevraždě, návody k nim, romantizace poruch příjmu potravy.",
        "— Spam a umělé navyšování: hromadné stejné zprávy, nevyžádaná reklama, farmy účtů, umělé navyšování reakcí a hlasů.",
        "— Podvody a klamání: phishing, vylákání peněz a hesel, falešné soutěže, škodlivé soubory a odkazy.",
        "— Vydávání se za jinou osobu nebo organizaci, včetně zavádějícího jména a avataru.",
        "— Prodej a propagace zakázaného zboží: drog, zbraní, padělaných dokladů, kradených věcí.",
        "— Porušování práv druhých: zveřejňování cizích děl, fotografií a materiálů bez oprávnění.",
        "— Obcházení omezení: nový účet k pokračování v tom, za co byl předchozí účet omezen nebo smazán.",
      ],
    },
    {
      title: "Skupiny a komunity",
      body: [
        "Ten, kdo skupinu nebo komunitu založil, a jím jmenovaní editoři odpovídají za pořádek uvnitř a mohou mazat příspěvky. Vnitřní pravidla komunity nemohou povolit to, co tato pravidla zakazují.",
        "Uzavřená komunita není místo, kde pravidla neplatí: uzavřenost chrání členy před cizíma očima, ne porušení před moderací.",
      ],
    },
    {
      title: "Soukromé konverzace",
      body: [
        "Soukromé chaty jsou šifrované na zařízeních a číst je nemůžeme — ani při nahlášení: u zprávy ze soukromého chatu vidíme jen to, že nahlášení existuje. Hlavní ochrana je tady proto ve tvých rukou: protějšek můžeš kdykoli zablokovat a o účtu se rozhoduje na základě souhrnu nahlášení.",
        "Šifrování nedělá ze zakázaného povolené. Pokud protějšek porušení ukáže sám — například snímkem obrazovky ve zprávě pro nás —, máme právo přijmout opatření podle těchto pravidel.",
      ],
    },
    {
      title: "Jak se chránit a nahlásit",
      body: [
        "— Zablokovat. Tlačítko v záhlaví soukromého chatu s daným člověkem. Blokování je jednostranné a tiché: dotyčný se o něm nedozví, nebude ti moct psát soukromé zprávy ani volat a jeho příspěvky v komunitách zmizí z tvého feedu. Blokování zrušíš v profilu → „Zablokovaní“.",
        "— Nahlásit můžeš zprávu (z její nabídky), příspěvek v komunitě, člověka (v záhlaví soukromého chatu nebo v seznamu členů skupiny) i samotnou komunitu nebo skupinu. Ten, koho nahlásíš, se to nedozví.",
        "— Nahlásit můžeš jen to, co vidíš na vlastní oči.",
        "— Příspěvek v komunitě, který nahlásí pět různých lidí, se skryje z feedu, dokud nebude zkontrolován. Nahlášení od účtů vytvořených v posledních hodinách se k administraci dostanou, ale do těch pěti se nepočítají — jinak by šlo příspěvek skrýt dávkou čerstvých účtů.",
        "— Nahlášení posuzuje administrace služby. Pokud ti někdo vyhrožuje nebo jsi v nebezpečí, obrať se nejdřív na policii: nejsme služba tísňové pomoci.",
      ],
    },
    {
      title: "Co hrozí za porušení",
      body: [
        "Podle závažnosti a opakování:",
        "— smazání příspěvku v komunitě — jejími editory nebo administrací;",
        "— u skupiny nebo komunity: varování, vyřazení z hlavního feedu, smazání (třetí platné varování komunitu smaže);",
        "— u účtu: zablokování, po kterém se k němu už nelze přihlásit.",
        "Za závažná porušení — ohrožení života, sexuální zneužívání dětí, terorismus — se účet blokuje bez varování a informace mohou být předány orgánům činným v trestním řízení postupem stanoveným zákonem.",
        "Účet vytvořený bez pozvánky funguje první hodiny po registraci v omezeném režimu: je to ochrana proti spamu, ne trest.",
      ],
    },
    {
      title: "Pokud nesouhlasíš s rozhodnutím",
      body: [
        "Napiš na kisyandco@gmail.com: uživatelské jméno, co se stalo a proč považuješ rozhodnutí za chybné. Rozhodnutí přezkoumáme a odpovíme, obvykle do 30 dnů.",
      ],
    },
    {
      title: "Změny pravidel",
      body: [
        "Pravidla se mohou měnit. Při podstatných změnách tě aplikace při příštím přihlášení požádá o přijetí nového znění — bez toho službu používat nelze. Datum poslední změny je uvedeno v horní části stránky.",
      ],
    },
  ],
};
