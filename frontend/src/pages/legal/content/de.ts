// Deutsche Übersetzung von content/ru.ts. Rechtlich verbindlich ist die russische Fassung; dies ist eine Übersetzung zur Information.
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-07",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "Kurz gesagt",
      body: [
        "KISY ist ein Messenger. Damit er funktioniert, speichern wir dein Konto, deine Chats und die Dateien, die du sendest. Wir verkaufen diese Daten nicht, zeigen keine Werbung und geben sie nicht an Dritte weiter – außer in den unten beschriebenen Fällen.",
        "Private Einzelchats werden auf den Geräten verschlüsselt: Der Server speichert sie in einer Form, die er selbst nicht lesen kann. Nachrichten in Gruppen und Beiträge in Communitys werden unverschlüsselt gespeichert – der Administrator des Servers kann sie sehen.",
        "Du kannst dein Konto jederzeit löschen: Profil → „Konto löschen“ oder auf der Seite zur Kontolöschung, ohne dich in der App anzumelden.",
      ],
    },
    {
      title: "Wer für die Daten verantwortlich ist",
      body: [
        "Der Dienst wird als privates Projekt betrieben. Verantwortlich für die Daten ist der Inhaber der KISY-Installation, bei der du dich registriert hast – er beantwortet auch Fragen zu deinen Daten unter der Adresse am Ende dieses Dokuments.",
        "Die App ist kostenlos und wird „wie besehen“ bereitgestellt, ohne Garantie für die Verfügbarkeit und den Erhalt der Daten (siehe Abschnitt „Haftung“).",
      ],
    },
    {
      title: "Welche Daten wir erheben",
      body: [
        "— Konto: Benutzername, Anzeigename, Passwort-Hash (das Passwort selbst wird nicht gespeichert), Registrierungsdatum, Zugriffsstufe, Profilbild, falls du eines hochgeladen hast.",
        "— Inhalte: Nachrichten, Dateien, Sprachnachrichten, Notizen, Beiträge in Communitys, Reaktionen, Kalender und Aufgaben – alles, was du in der App erstellst.",
        "— Technische Daten: Zeitpunkte der Anmeldungen, ein nicht umkehrbarer Fingerabdruck (Hash) der IP-Adresse, Name des Geräts und des Browsers, Kennungen für Push-Benachrichtigungen, das Protokoll der Administratoraktionen.",
        "— Anrufe: dass ein Anruf stattgefunden hat, sein Zeitpunkt und seine Dauer. Der Inhalt des Anrufs wird nicht aufgezeichnet.",
        "— Zustimmung: wann du diese Datenschutzerklärung und die Community-Richtlinien akzeptiert hast, welche Fassungen davon, und ein nicht umkehrbarer Fingerabdruck der IP-Adresse zu diesem Zeitpunkt. Das ist der Nachweis deiner Zustimmung, deshalb wird er auch nach der Löschung des Kontos aufbewahrt – genau wie das Sicherheitsprotokoll.",
        "Wir fragen deine Telefonnummer, Adresse, Zahlungsdaten und deinen genauen Standort nicht ab und speichern sie nicht.",
      ],
    },
    {
      title: "Wofür wir sie brauchen",
      body: [
        "— Um Nachrichten zuzustellen und deine Chats auf deinen Geräten anzuzeigen.",
        "— Um den Dienst zu schützen: Spam und das Durchprobieren von Passwörtern zu begrenzen und Missbrauch zu erkennen (genau dafür brauchen wir den Hash der IP-Adresse und nicht die Adresse selbst).",
        "— Um dir Benachrichtigungen zu senden, wenn du sie aktiviert hast.",
        "— Um gesetzliche Anforderungen zu erfüllen, soweit sie für uns gelten.",
        "Wir nutzen deine Daten nicht für Werbung, Profilbildung oder das Training von Modellen.",
      ],
    },
    {
      title: "Wer Zugriff auf die Daten erhält",
      body: [
        "— Andere Nutzer – genau die Personen, denen du schreibst, und die Mitglieder der Gruppen und Communitys, in denen du schreibst.",
        "— Infrastrukturanbieter, ohne die der Dienst nicht funktioniert: Hosting der App und der Datenbank, Dateispeicher, Zustelldienst für Push-Benachrichtigungen (Google Firebase), Schutz vor Bots (Cloudflare Turnstile). Sie verarbeiten die Daten in unserem Auftrag und nutzen sie nicht für eigene Zwecke.",
        "— Der Administrator des Servers – in dem Umfang, der im Abschnitt „Was der Administrator sieht“ beschrieben ist.",
        "Wir verkaufen keine Daten und geben sie nicht an Werbenetzwerke weiter.",
      ],
    },
    {
      title: "Was der Administrator sieht",
      body: [
        "Ehrlich zu den Grenzen der Verschlüsselung. Der Administrator des Servers kann private Einzelchats nicht lesen: Sie sind mit Schlüsseln verschlüsselt, die auf den Geräten bleiben.",
        "Der Administrator kann sehen: Nachrichten in Gruppenchats und Beiträge in Communitys, Namen und Benutzernamen, dass und wann geschrieben wurde (wer mit wem und wann), hochgeladene Dateien, das Aktionsprotokoll. Bei einer Meldung zu einer privaten Nachricht sehen wir nur, dass es eine Meldung gibt – der Text bleibt verschlüsselt.",
        "Wenn du vollständige Vertraulichkeit deiner Unterhaltungen brauchst, nutze private Chats und keine Gruppen.",
      ],
    },
    {
      title: "Wie lange wir Daten speichern",
      body: [
        "Nachrichten und Dateien werden gespeichert, bis du sie löschst oder dein Konto löschst. Selbstlöschende Nachrichten werden nach dem von dir eingestellten Timer gelöscht.",
        "Das Aktionsprotokoll und die Einträge zu Anmeldungen werden bis zu einem Jahr aufbewahrt – sie werden gebraucht, um Hackerangriffe aufzuklären.",
        "Sicherungskopien der Datenbank werden verschlüsselt bis zu 30 Tage aufbewahrt. Gelöschte Daten verschwinden aus den Kopien, sobald diese ablaufen.",
      ],
    },
    {
      title: "Kontolöschung",
      body: [
        "Du kannst dein Konto in der App löschen: Profil → „Konto löschen“, Bestätigung mit deinem Passwort. Dasselbe ist auf der Seite zur Kontolöschung möglich, ohne die App zu installieren.",
        "Die Löschung erfolgt sofort und kann nicht rückgängig gemacht werden. Gelöscht werden: Passwort und alle Sitzungen, Verschlüsselungsschlüssel, Push-Tokens, deine Dateien und Notizen, Einstellungen, Reaktionen und Stimmen, die Texte deiner privaten Nachrichten – auch bei der anderen Person.",
        "Erhalten bleiben: deine Nachrichten in Gruppenchats und Beiträge in Communitys (das sind Unterhaltungen anderer Menschen und der öffentliche Feed) – dann ohne Namen, als „Gelöschtes Konto“; das Sicherheitsprotokoll, das nicht nachträglich verändert werden darf.",
        "Gruppen und Communitys, die du verwaltet hast, gehen an die nächste Person in der Verwaltung über; gibt es niemanden, werden sie zusammen mit dem Konto gelöscht. Dein Benutzername wird nie an jemand anderen vergeben.",
      ],
    },
    {
      title: "Deine Rechte",
      body: [
        "Du kannst eine Kopie deiner Daten erhalten, sie berichtigen, dein Konto löschen oder der Verarbeitung widersprechen – schreib uns an die Adresse am Ende dieses Dokuments. Wir antworten innerhalb einer angemessenen Frist, in der Regel innerhalb von 30 Tagen.",
        "Wenn du dich in der EU oder im Vereinigten Königreich befindest, hast du das Recht, dich bei der Aufsichtsbehörde deines Landes zu beschweren.",
      ],
    },
    {
      title: "Sicherheit und ihre Grenzen",
      body: [
        "Was wir tun: Passwörter werden nur als Hashes gespeichert (Argon2id), private Chats werden auf den Geräten verschlüsselt, der Datenverkehr läuft über TLS, Aktionen der Administratoren werden protokolliert, Sicherungskopien werden verschlüsselt, und es gibt Schutz vor dem Erraten von Passwörtern und vor Spam-Registrierungen.",
        "Was wir nicht versprechen. Kein Dienst ist absolut sicher. Risiken, die du vorab kennen solltest:",
        "— Ein Angriff auf den Server oder die Infrastruktur eines Anbieters kann alles offenlegen außer dem Inhalt privater Chats.",
        "— Zugriff auf dein Gerät (Diebstahl, Schadsoftware, fremde Hände) legt auch private Chats offen: Die Schlüssel sind auf dem Gerät gespeichert.",
        "— Die andere Person kann speichern, weiterleiten oder abfotografieren, was du ihr geschickt hast. Technisch lässt sich das nicht verhindern.",
        "— Der Verlust des Geräts oder eine Neuinstallation der App kann bedeuten, dass der Verlauf privater Chats verloren geht: Die Verschlüsselungsschlüssel verlassen das Gerät nicht, und wir können sie nicht für dich wiederherstellen.",
        "— Ein Ausfall des Hostings, ein Programmfehler oder das Ausschöpfen des kostenlosen Tarifs können dazu führen, dass der Dienst nicht verfügbar ist und Daten verloren gehen. Sichere Wichtiges selbst.",
        "— Push-Benachrichtigungen laufen über Google und werden auf dem Weg zum Gerät nicht von uns verschlüsselt; sie können den Namen des Absenders und den Anfang der Nachricht enthalten, wenn du die Vorschau nicht deaktiviert hast.",
      ],
    },
    {
      title: "Kinder",
      body: [
        "Der Dienst ist nicht für Kinder unter 13 Jahren bestimmt (in der EU unter 16 Jahren oder unter dem in deinem Land festgelegten Alter). Wenn du erfährst, dass ein Kind den Dienst ohne Zustimmung der Eltern nutzt, schreib uns, und wir löschen das Konto.",
      ],
    },
    {
      title: "Haftung",
      body: [
        "Der Dienst wird „wie besehen“ und „wie verfügbar“ bereitgestellt, ohne jegliche Garantien: für Verfügbarkeit, Erhalt der Daten, Eignung für einen bestimmten Zweck und Fehlerfreiheit.",
        "Im größtmöglichen nach geltendem Recht zulässigen Umfang haftet der Inhaber des Dienstes nicht für: Verlust oder Beschädigung von Daten und Chats, Nichtverfügbarkeit des Dienstes, die Unmöglichkeit, Verschlüsselungsschlüssel und Verlauf wiederherzustellen, von Nutzern erstellte Inhalte, Handlungen anderer Nutzer und Dritter sowie für indirekte Schäden, entgangenen Gewinn und jegliche Folgen der Nutzung oder Unmöglichkeit der Nutzung des Dienstes.",
        "Die Inhalte der Chats erstellen die Nutzer. Der Inhaber des Dienstes prüft sie nicht vorab und haftet nicht für sie; bei Verstößen gegen die Richtlinien kann ein Konto eingeschränkt oder gelöscht werden.",
        "Die obigen Vorbehalte heben nicht auf, was nach dem Gesetz nicht ausgeschlossen werden kann: die Haftung für Vorsatz und grobe Fahrlässigkeit, für Schäden an Leben und Gesundheit sowie die Verbraucherrechte und die Pflichten des Verantwortlichen für personenbezogene Daten in deinem Land. Wo eine solche Beschränkung unzulässig ist, gilt sie nur in dem geringsten gesetzlich zulässigen Umfang.",
      ],
    },
    {
      title: "Änderungen",
      body: [
        "Wir können diese Datenschutzerklärung ändern. Wesentliche Änderungen werden in der App angezeigt. Das Datum der letzten Änderung steht oben auf der Seite.",
      ],
    },
    {
      title: "Kontakt",
      body: [
        "Fragen zu Daten, Löschanfragen und Beschwerden: kisyandco@gmail.com. Wir antworten innerhalb einer angemessenen Frist, in der Regel innerhalb von 30 Tagen.",
      ],
    },
  ],
  deletion: [
    {
      title: "So löschst du dein Konto in der App",
      body: [
        "1. Öffne KISY und melde dich bei deinem Konto an.",
        "2. Profil (Symbol unten rechts) → „Konto löschen“.",
        "3. Gib dein Passwort und das Wort LÖSCHEN ein.",
        "4. Fertig: Das Konto wird sofort gelöscht, rückgängig machen lässt sich das nicht.",
      ],
    },
    {
      title: "Was gelöscht wird",
      body: [
        "— Passwort, alle aktiven Sitzungen und Geräte.",
        "— Verschlüsselungsschlüssel und Push-Tokens.",
        "— Deine Dateien, Notizen, Einstellungen, Reaktionen und Stimmen.",
        "— Die Texte deiner privaten Nachrichten, auch bei der anderen Person.",
      ],
    },
    {
      title: "Was erhalten bleibt",
      body: [
        "— Deine Nachrichten in Gruppenchats und Beiträge in Communitys – ohne Namen, als „Gelöschtes Konto“. Das sind Unterhaltungen anderer Menschen und der öffentliche Feed, und wir können sie nicht für andere löschen.",
        "— Das Sicherheitsprotokoll (wer sich wann angemeldet hat, Aktionen der Administratoren) – bis zu einem Jahr, ohne Inhalte von Chats.",
        "— Der Eintrag darüber, dass du die Datenschutzerklärung und die Community-Richtlinien akzeptiert hast: wann und welche Fassungen davon. Ohne ihn lässt sich nicht nachweisen, dass eine Zustimmung vorlag.",
        "— Verschlüsselte Sicherungskopien der Datenbank – bis zu 30 Tage, danach verschwinden sie.",
      ],
    },
    {
      title: "Wenn du dich nicht anmelden kannst",
      body: [
        "Schreib an kisyandco@gmail.com und nenne den Benutzernamen, den du löschen möchtest. Wir löschen das Konto, nachdem wir geprüft haben, dass es dir gehört.",
      ],
    },
  ],
  rules: [
    {
      title: "Kurz gesagt",
      body: [
        "KISY ist ein Ort für Unterhaltungen, gemeinsame Gruppen und Communitys. Die folgenden Richtlinien gelten überall, wo du etwas schreibst, hochlädst oder anderen zeigst: in privaten Chats und Gruppenchats, in Communitys und ihrem Feed, in Namen und Beschreibungen von Gruppen, im Namen und Profilbild deines Profils.",
        "Die wichtigste Regel: Tu anderen nichts an, wofür man im normalen Leben vor dem Gesetz oder vor anderen Menschen geradestehen muss. Wenn du unsicher bist – veröffentliche es nicht.",
      ],
    },
    {
      title: "Was verboten ist",
      body: [
        "— Sexueller Missbrauch von Kindern und jegliches Material, das Minderjährige sexualisiert. Hier gibt es keine Verwarnungen: Das Konto wird sofort gesperrt, und die Informationen werden an die Strafverfolgungsbehörden weitergegeben.",
        "— Explizite sexuelle Inhalte und Pornografie. Der Dienst richtet sich an Nutzer ab 13 Jahren.",
        "— Drohungen, Aufrufe zu Gewalt, Verherrlichung von Gewalt und Terrorismus, Anwerbung für extremistische Organisationen.",
        "— Mobbing und Belästigung: Beleidigungen, systematische Angriffe auf eine Person, das Aufhetzen anderer, wiederholte Nachrichten an jemanden, der dich blockiert oder dich gebeten hat aufzuhören.",
        "— Hetze und Herabwürdigung von Menschen aufgrund von Nationalität, ethnischer Zugehörigkeit, Religion, Geschlecht, sexueller Orientierung, Behinderung, Alter oder Herkunft.",
        "— Veröffentlichung personenbezogener Daten anderer ohne deren Zustimmung: Adresse, Telefonnummer, Dokumente, Fotos aus dem Privatleben, Chats.",
        "— Aufrufe zu Selbstverletzung und Suizid, Anleitungen dazu, Verherrlichung von Essstörungen.",
        "— Spam und Manipulation: massenhaft gleiche Nachrichten, unerwünschte Werbung, Kontofarmen, künstliches Hochtreiben von Reaktionen und Stimmen.",
        "— Betrug und Täuschung: Phishing, Erschleichen von Geld und Passwörtern, gefälschte Gewinnspiele, schädliche Dateien und Links.",
        "— Sich als eine andere Person oder Organisation ausgeben, auch durch irreführende Namen und Profilbilder.",
        "— Verkauf von und Werbung für Verbotenes: Drogen, Waffen, gefälschte Dokumente, Diebesgut.",
        "— Verletzung fremder Rechte: Veröffentlichung fremder Werke, Fotos und Materialien ohne das Recht dazu.",
        "— Umgehung von Beschränkungen: ein neues Konto, um fortzusetzen, wofür das bisherige eingeschränkt oder gelöscht wurde.",
      ],
    },
    {
      title: "Gruppen und Communitys",
      body: [
        "Wer eine Gruppe oder Community erstellt hat, und die von dieser Person ernannten Redakteure sind für die Ordnung darin verantwortlich und können Beiträge löschen. Interne Regeln einer Community können nicht erlauben, was diese Richtlinien verbieten.",
        "Eine geschlossene Community ist kein Ort, an dem die Richtlinien nicht gelten: Die Geschlossenheit schützt die Mitglieder vor fremden Blicken, nicht Verstöße vor der Moderation.",
      ],
    },
    {
      title: "Private Chats",
      body: [
        "Private Chats werden auf den Geräten verschlüsselt, und wir können sie nicht lesen – auch nicht bei einer Meldung: Bei einer Nachricht aus einem privaten Chat sehen wir nur, dass sie gemeldet wurde. Deshalb liegt der wichtigste Schutz hier bei dir: Du kannst die andere Person jederzeit blockieren, und Entscheidungen über ein Konto werden anhand der Gesamtheit der Meldungen getroffen.",
        "Verschlüsselung macht Verbotenes nicht erlaubt. Wenn die andere Person uns einen Verstoß selbst zeigt – zum Beispiel per Screenshot in einer Nachricht an uns –, dürfen wir nach diesen Richtlinien Maßnahmen ergreifen.",
      ],
    },
    {
      title: "So schützt du dich und meldest etwas",
      body: [
        "— Blockieren. Die Schaltfläche befindet sich oben im privaten Chat mit der Person. Die Blockierung ist einseitig und still: Die Person erfährt nichts davon, kann dir keine privaten Nachrichten mehr schreiben und dich nicht anrufen, und ihre Beiträge in Communitys verschwinden aus deinem Feed. Aufheben kannst du die Blockierung unter Profil → „Blockiert“.",
        "— Melden kannst du eine Nachricht (über ihr Menü), einen Beitrag in einer Community, eine Person (oben im privaten Chat oder in der Mitgliederliste einer Gruppe) und die Community oder Gruppe selbst. Wen du meldest, erfährt davon nichts.",
        "— Melden kannst du nur, was du selbst siehst.",
        "— Ein Beitrag in einer Community, den fünf verschiedene Personen gemeldet haben, wird aus dem Feed ausgeblendet, bis er geprüft wurde. Meldungen von Konten, die in den letzten Stunden erstellt wurden, erreichen die Administration, zählen aber nicht zu diesen fünf – sonst ließe sich ein Beitrag mit einem Schwung frischer Konten ausblenden.",
        "— Meldungen prüft die Administration des Dienstes. Wenn du bedroht wirst oder in Gefahr bist, wende dich zuerst an die Polizei: Wir sind kein Notdienst.",
      ],
    },
    {
      title: "Was bei Verstößen passiert",
      body: [
        "Je nach Schwere und Wiederholung:",
        "— Löschung eines Beitrags in einer Community – durch ihre Redakteure oder die Administration;",
        "— für eine Gruppe oder Community: Verwarnung, Ausschluss aus dem allgemeinen Feed, Löschung (die dritte gültige Verwarnung löscht die Community);",
        "— für ein Konto: Sperrung, danach ist keine Anmeldung mehr möglich.",
        "Bei schweren Verstößen – Bedrohung des Lebens, sexueller Missbrauch von Kindern, Terrorismus – wird das Konto ohne Verwarnung gesperrt, und Informationen können im gesetzlich vorgesehenen Verfahren an die Strafverfolgungsbehörden weitergegeben werden.",
        "Ein Konto, das ohne Einladung erstellt wurde, ist in den ersten Stunden nach der Registrierung eingeschränkt: Das ist ein Schutz vor Spam, keine Strafe.",
      ],
    },
    {
      title: "Wenn du mit einer Entscheidung nicht einverstanden bist",
      body: [
        "Schreib an kisyandco@gmail.com: deinen Benutzernamen, was passiert ist und warum du die Entscheidung für falsch hältst. Wir prüfen sie erneut und antworten, in der Regel innerhalb von 30 Tagen.",
      ],
    },
    {
      title: "Änderungen der Richtlinien",
      body: [
        "Die Richtlinien können sich ändern. Bei wesentlichen Änderungen bittet dich die App bei der nächsten Anmeldung, die neue Fassung zu akzeptieren – ohne das kannst du den Dienst nicht nutzen. Das Datum der letzten Änderung steht oben auf der Seite.",
      ],
    },
  ],
};
