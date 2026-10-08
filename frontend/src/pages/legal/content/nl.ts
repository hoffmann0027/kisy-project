// Nederlandse vertaling van content/ru.ts. Alleen de Russische versie is juridisch bindend; dit is een vertaling voor het gemak.
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-07",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "In het kort",
      body: [
        "KISY is een messenger. Om die te laten werken, bewaren we je account, je gesprekken en de bestanden die je verstuurt. We verkopen deze gegevens niet, tonen geen advertenties en geven ze niet door aan derden, behalve in de gevallen die hieronder worden beschreven.",
        "Privégesprekken één-op-één zijn op de apparaten versleuteld: de server bewaart ze in een vorm die hij zelf niet kan lezen. Berichten in groepen en posts in community's worden onversleuteld bewaard — de beheerder van de server kan ze zien.",
        "Je kunt je account op elk moment verwijderen: Profiel → “Account verwijderen”, of op de pagina voor het verwijderen van je account, zonder in de app in te loggen.",
      ],
    },
    {
      title: "Wie verantwoordelijk is voor de gegevens",
      body: [
        "De dienst draait als privéproject. Verwerkingsverantwoordelijke is de eigenaar van de KISY-installatie waarop je je hebt geregistreerd — die beantwoordt ook vragen over je gegevens via het adres aan het einde van dit document.",
        "De app is gratis en wordt verspreid “zoals hij is”, zonder garanties voor de beschikbaarheid en het behoud van gegevens (zie het onderdeel “Aansprakelijkheid”).",
      ],
    },
    {
      title: "Welke gegevens we verzamelen",
      body: [
        "— Account: gebruikersnaam, weergavenaam, hash van je wachtwoord (het wachtwoord zelf wordt niet bewaard), registratiedatum, toegangsniveau en je avatar, als je die hebt geüpload.",
        "— Inhoud: berichten, bestanden, spraakberichten, notities, posts in community's, reacties, agenda en taken — alles wat je in de app maakt.",
        "— Technische gegevens: tijdstippen waarop je inlogt, een onomkeerbare vingerafdruk (hash) van je IP-adres, de naam van je apparaat en browser, identificatoren voor pushmeldingen, het logboek van acties van beheerders.",
        "— Oproepen: het feit dat er een oproep was, het tijdstip en de duur. De inhoud van een oproep wordt niet opgenomen.",
        "— Toestemming: wanneer je dit beleid en de communityregels hebt geaccepteerd, welke versies daarvan, en de onomkeerbare vingerafdruk van je IP-adres op dat moment. Dit is het bewijs van je toestemming en wordt daarom ook na het verwijderen van je account bewaard — net als het beveiligingslogboek.",
        "We vragen en bewaren geen telefoonnummer, adres, betaalgegevens of exacte locatie.",
      ],
    },
    {
      title: "Waarvoor we dit nodig hebben",
      body: [
        "— Om berichten af te leveren en gesprekken op je apparaten te tonen.",
        "— Om de dienst te beschermen: spam en het raden van wachtwoorden beperken, misbruik opsporen (daarvoor is de hash van het IP-adres nodig, niet het adres zelf).",
        "— Om je meldingen te sturen, als je die hebt aangezet.",
        "— Om te voldoen aan wettelijke eisen, wanneer die op ons van toepassing zijn.",
        "We gebruiken je gegevens niet voor advertenties, profilering of het trainen van modellen.",
      ],
    },
    {
      title: "Wie toegang krijgt tot de gegevens",
      body: [
        "— Andere gebruikers — precies degene naar wie je schrijft, en de leden van de groepen en community's waarin je schrijft.",
        "— Leveranciers van infrastructuur zonder wie de dienst niet werkt: hosting van de app en de database, bestandsopslag, de dienst die pushmeldingen aflevert (Google Firebase) en bescherming tegen bots (Cloudflare Turnstile). Zij verwerken de gegevens in onze opdracht en gebruiken ze niet voor eigen doeleinden.",
        "— De beheerder van de server — in de mate die wordt beschreven in het onderdeel “Wat de beheerder ziet”.",
        "We verkopen geen gegevens en geven ze niet door aan advertentienetwerken.",
      ],
    },
    {
      title: "Wat de beheerder ziet",
      body: [
        "Eerlijk over de grenzen van versleuteling. De beheerder van de server kan privégesprekken één-op-één niet lezen: die zijn versleuteld met sleutels die op de apparaten blijven.",
        "De beheerder kan zien: berichten in groepschats en posts in community's, namen en gebruikersnamen, het feit en het tijdstip van gesprekken (wie met wie en wanneer), geüploade bestanden en het logboek van acties. Als een privébericht wordt gerapporteerd, zien we alleen dat er een klacht is — de tekst blijft versleuteld.",
        "Wil je volledige vertrouwelijkheid van je gesprekken, gebruik dan privéchats en geen groepen.",
      ],
    },
    {
      title: "Hoe lang we gegevens bewaren",
      body: [
        "Berichten en bestanden worden bewaard totdat je ze verwijdert of totdat je je account verwijdert. Verdwijnende berichten worden verwijderd volgens de timer die je hebt ingesteld.",
        "Het logboek van acties en de gegevens over inloggen worden maximaal een jaar bewaard — ze zijn nodig om inbraken te onderzoeken.",
        "Back-ups van de database worden versleuteld en maximaal 30 dagen bewaard. Verwijderde gegevens verdwijnen uit de back-ups naarmate die verouderen.",
      ],
    },
    {
      title: "Account verwijderen",
      body: [
        "Je kunt je account in de app verwijderen: Profiel → “Account verwijderen”, met bevestiging via je wachtwoord. Hetzelfde kan op de pagina voor het verwijderen van je account, zonder de app te installeren.",
        "Het verwijderen gebeurt direct en kan niet ongedaan worden gemaakt. Verwijderd worden: je wachtwoord en alle sessies, versleutelingssleutels, push-tokens, je bestanden en notities, instellingen, reacties en stemmen, en de tekst van je privéberichten — ook bij je gesprekspartner.",
        "Blijven bewaard: je berichten in groepschats en posts in community's (dit zijn gesprekken van anderen en een openbare feed) — zonder je naam, afkomstig van “Verwijderd account”; en het beveiligingslogboek, dat niet met terugwerkende kracht mag worden herschreven.",
        "Groepen en community's die je beheerde, gaan over naar de volgende beheerder; is die er niet, dan worden ze samen met je account verwijderd. Je gebruikersnaam gaat nooit naar iemand anders.",
      ],
    },
    {
      title: "Je rechten",
      body: [
        "Je kunt een kopie van je gegevens krijgen, ze corrigeren, je account verwijderen of bezwaar maken tegen de verwerking — schrijf ons via het adres aan het einde van dit document. We antwoorden binnen een redelijke termijn, meestal binnen 30 dagen.",
        "Als je je in de EU of het Verenigd Koninkrijk bevindt, heb je het recht een klacht in te dienen bij de toezichthouder in je land.",
      ],
    },
    {
      title: "Beveiliging en de grenzen ervan",
      body: [
        "Wat we doen: wachtwoorden worden alleen als hash bewaard (Argon2id), privégesprekken worden op de apparaten versleuteld, het verkeer loopt via TLS, acties van beheerders worden gelogd, back-ups worden versleuteld, en er is bescherming tegen het raden van wachtwoorden en tegen spamregistraties.",
        "Wat we niet beloven. Geen enkele dienst is volledig beveiligd. Risico's die je vooraf moet kennen:",
        "— Een inbraak op de server of in de infrastructuur van een leverancier kan alles blootleggen, behalve de inhoud van privégesprekken.",
        "— Toegang tot je apparaat (diefstal, malware, andermans handen) legt ook je privégesprekken bloot: de sleutels staan op het apparaat.",
        "— Je gesprekspartner kan bewaren, doorsturen of fotograferen wat je hebt verstuurd. Technisch is dat niet te voorkomen.",
        "— Als je je apparaat verliest of de app opnieuw installeert, kan dat betekenen dat je de geschiedenis van je privégesprekken kwijtraakt: de versleutelingssleutels verlaten het apparaat niet en wij kunnen ze niet voor je herstellen.",
        "— Een storing bij de hosting, een fout in de software of het opraken van een gratis abonnement kan ertoe leiden dat de dienst onbereikbaar wordt en dat gegevens verloren gaan. Maak zelf back-ups van wat belangrijk voor je is.",
        "— Pushmeldingen lopen via Google en worden onderweg naar je apparaat niet door ons versleuteld; ze kunnen de naam van de afzender en het begin van het bericht bevatten, als je het voorbeeld van de tekst niet hebt uitgezet.",
      ],
    },
    {
      title: "Kinderen",
      body: [
        "De dienst is niet bedoeld voor kinderen jonger dan 13 jaar (in de EU: jonger dan 16 jaar of de leeftijd die in je land geldt). Als je ontdekt dat een kind de dienst zonder toestemming van de ouders gebruikt, schrijf ons dan, en we verwijderen het account.",
      ],
    },
    {
      title: "Aansprakelijkheid",
      body: [
        "De dienst wordt geleverd “zoals hij is” en “zoals beschikbaar”, zonder enige garantie: niet voor de beschikbaarheid, het behoud van gegevens, de geschiktheid voor een bepaald doel of de afwezigheid van fouten.",
        "Voor zover maximaal toegestaan door het toepasselijke recht is de eigenaar van de dienst niet aansprakelijk voor: verlies of beschadiging van gegevens en gesprekken, onbeschikbaarheid van de dienst, de onmogelijkheid om versleutelingssleutels en geschiedenis te herstellen, inhoud die door gebruikers is gemaakt, handelingen van andere gebruikers en derden, en evenmin voor indirecte schade, gederfde winst en alle gevolgen van het gebruik of het niet kunnen gebruiken van de dienst.",
        "De inhoud van gesprekken wordt door gebruikers gemaakt. De eigenaar van de dienst controleert die niet vooraf en is er niet verantwoordelijk voor; bij overtreding van de regels kan een account worden beperkt of verwijderd.",
        "De voorbehouden hierboven doen niets af aan wat volgens de wet niet kan worden uitgesloten: aansprakelijkheid voor opzet en grove nalatigheid en voor schade aan leven en gezondheid, en evenmin aan de rechten van consumenten en de plichten van de verwerkingsverantwoordelijke voor persoonsgegevens in je land. Waar een dergelijke beperking niet is toegestaan, wordt ze toegepast in de minimale mate die de wet toestaat.",
      ],
    },
    {
      title: "Wijzigingen",
      body: [
        "We kunnen dit beleid wijzigen. Belangrijke wijzigingen worden in de app getoond. De datum van de laatste wijziging staat bovenaan de pagina.",
      ],
    },
    {
      title: "Contact",
      body: [
        "Vragen over gegevens, verzoeken om verwijdering en klachten: kisyandco@gmail.com. We antwoorden binnen een redelijke termijn, meestal binnen 30 dagen.",
      ],
    },
  ],
  deletion: [
    {
      title: "Je account verwijderen in de app",
      body: [
        "1. Open KISY en log in op je account.",
        "2. Profiel (icoon rechtsonder) → “Account verwijderen”.",
        "3. Voer je wachtwoord en het woord VERWIJDEREN in.",
        "4. Klaar: het account wordt direct verwijderd, ongedaan maken kan niet.",
      ],
    },
    {
      title: "Wat er wordt verwijderd",
      body: [
        "— Je wachtwoord, alle actieve sessies en apparaten.",
        "— Versleutelingssleutels en push-tokens.",
        "— Je bestanden, notities, instellingen, reacties en stemmen.",
        "— De tekst van je privéberichten, ook bij je gesprekspartner.",
      ],
    },
    {
      title: "Wat er blijft",
      body: [
        "— Je berichten in groepschats en posts in community's — zonder naam, afkomstig van “Verwijderd account”. Dit zijn gesprekken van anderen en een openbare feed, en die kunnen we niet namens anderen wissen.",
        "— Het beveiligingslogboek (wie wanneer heeft ingelogd, acties van beheerders) — maximaal een jaar, zonder de inhoud van gesprekken.",
        "— De vastlegging dat je het beleid en de communityregels hebt geaccepteerd: wanneer en welke versies. Zonder die vastlegging kan niet worden bewezen dat er toestemming was.",
        "— Versleutelde back-ups van de database — maximaal 30 dagen, daarna verdwijnen ze.",
      ],
    },
    {
      title: "Als inloggen niet lukt",
      body: [
        "Schrijf naar kisyandco@gmail.com en vermeld de gebruikersnaam die je wilt laten verwijderen. We verwijderen het account nadat we hebben gecontroleerd dat het van jou is.",
      ],
    },
  ],
  rules: [
    {
      title: "In het kort",
      body: [
        "KISY is een plek voor gesprekken, gedeelde groepen en community's. De regels hieronder gelden overal waar je iets schrijft, uploadt of aan anderen laat zien: in privé- en groepschats, in community's en hun feed, in namen en beschrijvingen van groepen, en in de naam en avatar van je profiel.",
        "De belangrijkste regel: doe anderen niets aan waarvoor je in het gewone leven verantwoording moet afleggen tegenover de wet of tegenover mensen. Twijfel je — publiceer het dan niet.",
      ],
    },
    {
      title: "Wat verboden is",
      body: [
        "— Seksueel misbruik van kinderen en elk materiaal dat minderjarigen seksualiseert. Hier zijn geen waarschuwingen: het account wordt direct geblokkeerd en de informatie wordt doorgegeven aan de opsporingsinstanties.",
        "— Expliciete seksuele inhoud en pornografie. De dienst is bedoeld voor gebruikers vanaf 13 jaar.",
        "— Bedreigingen, oproepen tot geweld, verheerlijking van geweld en terrorisme, werving voor extremistische organisaties.",
        "— Pesten en intimidatie: beledigingen, stelselmatige aanvallen op een persoon, anderen tegen iemand opzetten, herhaaldelijk berichten sturen aan iemand die je heeft geblokkeerd of je heeft gevraagd te stoppen.",
        "— Aanzetten tot haat en het vernederen van mensen op grond van nationaliteit, ras, religie, geslacht, seksuele geaardheid, handicap, leeftijd of afkomst.",
        "— Het publiceren van persoonsgegevens van anderen zonder hun toestemming: adres, telefoonnummer, documenten, foto's uit het privéleven, gesprekken.",
        "— Oproepen tot zelfbeschadiging en zelfdoding, instructies daarvoor, het romantiseren van eetstoornissen.",
        "— Spam en manipulatie: massaal dezelfde berichten, ongevraagde reclame, accountfarms, het kunstmatig opdrijven van reacties en stemmen.",
        "— Oplichting en bedrog: phishing, het aftroggelen van geld en wachtwoorden, nepwinacties, schadelijke bestanden en links.",
        "— Je voordoen als een andere persoon of organisatie, ook met een misleidende naam of avatar.",
        "— Verkoop van en reclame voor verboden zaken: drugs, wapens, valse documenten, gestolen goederen.",
        "— Inbreuk op de rechten van anderen: werken, foto's en materiaal van anderen publiceren zonder dat je daar het recht toe hebt.",
        "— Het omzeilen van beperkingen: een nieuw account om door te gaan met datgene waarvoor je vorige account is beperkt of verwijderd.",
      ],
    },
    {
      title: "Groepen en community's",
      body: [
        "Wie een groep of community heeft gemaakt, en de redacteuren die diegene heeft aangewezen, zijn verantwoordelijk voor de orde daarbinnen en kunnen posts verwijderen. De interne regels van een community kunnen niet toestaan wat door deze regels is verboden.",
        "Een besloten community is geen plek waar de regels niet gelden: beslotenheid beschermt de leden tegen de blikken van buitenstaanders, niet overtredingen tegen moderatie.",
      ],
    },
    {
      title: "Privégesprekken",
      body: [
        "Privéchats zijn op de apparaten versleuteld en wij kunnen ze niet lezen — ook niet als er iets wordt gerapporteerd: bij een bericht uit een privéchat zien we alleen dat er een klacht is. Daarom ligt de belangrijkste bescherming hier bij jou: je kunt je gesprekspartner op elk moment blokkeren, en besluiten over een account worden genomen op basis van alle klachten samen.",
        "Versleuteling maakt wat verboden is niet toegestaan. Als je gesprekspartner zelf een overtreding laat zien — bijvoorbeeld met een screenshot in een bericht aan ons — mogen we maatregelen nemen volgens deze regels.",
      ],
    },
    {
      title: "Jezelf beschermen en rapporteren",
      body: [
        "— Blokkeren. De knop staat bovenaan de privéchat met die persoon. Een blokkering is eenzijdig en stil: de ander merkt er niets van, kan je geen privéberichten meer sturen en je niet meer bellen, en de posts van die persoon in community's verdwijnen uit je feed. Je kunt de blokkering opheffen in je profiel → “Geblokkeerd”.",
        "— Rapporteren kan bij een bericht (via het menu ervan), een post in een community, een persoon (bovenaan de privéchat of in de ledenlijst van een groep) en bij de community of groep zelf. Degene die je rapporteert, komt dat niet te weten.",
        "— Je kunt alleen rapporteren wat je zelf ziet.",
        "— Een post in een community die door vijf verschillende mensen is gerapporteerd, wordt uit de feed verborgen totdat hij is gecontroleerd. Klachten van accounts die in de afgelopen uren zijn aangemaakt, komen wel bij het beheer terecht, maar tellen niet mee voor die vijf — anders zou je een post kunnen laten verbergen met een stel verse accounts.",
        "— Klachten worden beoordeeld door het beheer van de dienst. Word je bedreigd of ben je in gevaar, neem dan eerst contact op met de politie: wij zijn geen hulpdienst.",
      ],
    },
    {
      title: "Wat er gebeurt bij overtredingen",
      body: [
        "Afhankelijk van de ernst en of het vaker gebeurt:",
        "— verwijdering van een post in een community — door de redacteuren ervan of door het beheer;",
        "— voor een groep of community: een waarschuwing, uitsluiting van de algemene feed, verwijdering (de derde geldende waarschuwing verwijdert de community);",
        "— voor een account: een blokkering, waarna je niet meer kunt inloggen.",
        "Bij ernstige overtredingen — bedreiging van het leven, seksueel misbruik van kinderen, terrorisme — wordt het account zonder waarschuwing geblokkeerd, en kan de informatie volgens de wettelijke procedure worden doorgegeven aan de opsporingsinstanties.",
        "Een account dat zonder uitnodiging is aangemaakt, werkt de eerste uren na de registratie in een beperkte modus: dat is bescherming tegen spam, geen straf.",
      ],
    },
    {
      title: "Als je het niet eens bent met een besluit",
      body: [
        "Schrijf naar kisyandco@gmail.com: je gebruikersnaam, wat er is gebeurd en waarom je het besluit onjuist vindt. We heroverwegen het en antwoorden, meestal binnen 30 dagen.",
      ],
    },
    {
      title: "Wijzigingen van de regels",
      body: [
        "De regels kunnen veranderen. Bij belangrijke wijzigingen vraagt de app je bij de volgende keer inloggen om de nieuwe versie te accepteren — zonder dat kun je de dienst niet gebruiken. De datum van de laatste wijziging staat bovenaan de pagina.",
      ],
    },
  ],
};
