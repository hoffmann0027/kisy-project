// Polskie tłumaczenie content/ru.ts. Wiążąca prawnie jest wersja rosyjska; to tłumaczenie ma charakter pomocniczy.
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-07",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "W skrócie",
      body: [
        "KISY to komunikator. Aby działał, przechowujemy twoje konto, twoje rozmowy i pliki, które wysyłasz. Nie sprzedajemy tych danych, nie wyświetlamy reklam i nie przekazujemy ich osobom trzecim, z wyjątkiem przypadków opisanych poniżej.",
        "Prywatne rozmowy jeden na jeden są szyfrowane na urządzeniach: serwer przechowuje je w postaci, której sam nie może odczytać. Wiadomości w grupach i posty w społecznościach są przechowywane w postaci jawnej — widzi je administrator serwera.",
        "Możesz usunąć konto w dowolnym momencie: profil → „Usuń konto” albo na stronie usuwania konta, bez logowania się do aplikacji.",
      ],
    },
    {
      title: "Kto odpowiada za dane",
      body: [
        "Serwis działa jako prywatny projekt. Administratorem danych osobowych jest właściciel instalacji KISY, w której masz konto — to on odpowiada na pytania dotyczące twoich danych pod adresem podanym na końcu dokumentu.",
        "Aplikacja jest bezpłatna i udostępniana „tak jak jest”, bez gwarancji dostępności i zachowania danych (zob. sekcja „Odpowiedzialność”).",
      ],
    },
    {
      title: "Jakie dane zbieramy",
      body: [
        "— Konto: login, wyświetlane imię, hash hasła (samo hasło nie jest przechowywane), data rejestracji, poziom dostępu, awatar, jeśli został dodany.",
        "— Treści: wiadomości, pliki, wiadomości głosowe, notatki, posty w społecznościach, reakcje, kalendarz i zadania — wszystko, co tworzysz w aplikacji.",
        "— Dane techniczne: czas logowań, nieodwracalny odcisk (hash) adresu IP, nazwa urządzenia i przeglądarki, identyfikatory powiadomień push, dziennik działań administratorów.",
        "— Połączenia: fakt, czas i długość połączenia. Treść połączenia nie jest nagrywana.",
        "— Zgoda: kiedy zaakceptowano tę politykę i zasady społeczności, w jakich wersjach, oraz nieodwracalny odcisk adresu IP w tym momencie. To dowód zgody, dlatego jest przechowywany także po usunięciu konta — tak samo jak dziennik bezpieczeństwa.",
        "Nie prosimy o numer telefonu, adres, dane płatnicze ani dokładną lokalizację i nie przechowujemy ich.",
      ],
    },
    {
      title: "Do czego są potrzebne",
      body: [
        "— Aby dostarczać wiadomości i wyświetlać rozmowy na twoich urządzeniach.",
        "— Aby chronić serwis: ograniczać spam i próby odgadywania haseł, wykrywać nadużycia (właśnie do tego służy hash adresu IP, a nie sam adres).",
        "— Aby wysyłać powiadomienia, jeśli są włączone.",
        "— Aby spełniać wymogi prawa, gdy mają do nas zastosowanie.",
        "Nie wykorzystujemy twoich danych do reklam, profilowania ani trenowania modeli.",
      ],
    },
    {
      title: "Komu udostępniamy dane",
      body: [
        "— Innym użytkownikom — dokładnie tym, do których piszesz, oraz członkom grup i społeczności, w których piszesz.",
        "— Dostawcom infrastruktury, bez których serwis nie działa: hosting aplikacji i bazy danych, magazyn plików, usługa dostarczania powiadomień push (Google Firebase), ochrona przed botami (Cloudflare Turnstile). Przetwarzają oni dane na nasze zlecenie i nie wykorzystują ich do własnych celów.",
        "— Administratorowi serwera — w zakresie opisanym w sekcji „Co widzi administrator”.",
        "Nie sprzedajemy danych i nie przekazujemy ich sieciom reklamowym.",
      ],
    },
    {
      title: "Co widzi administrator",
      body: [
        "Uczciwie o granicach szyfrowania. Administrator serwera nie może odczytać prywatnych rozmów jeden na jeden: są one zaszyfrowane kluczami, które pozostają na urządzeniach.",
        "Administrator może zobaczyć: wiadomości w czatach grupowych i posty w społecznościach, imiona i loginy, fakt i czas korespondencji (kto z kim i kiedy), przesłane pliki, dziennik działań. Przy zgłoszeniu wiadomości prywatnej widzimy tylko fakt zgłoszenia — treść pozostaje zaszyfrowana.",
        "Jeśli potrzebujesz pełnej tajemnicy korespondencji, korzystaj z czatów prywatnych, a nie z grup.",
      ],
    },
    {
      title: "Jak długo przechowujemy dane",
      body: [
        "Wiadomości i pliki są przechowywane, dopóki ich nie usuniesz lub dopóki nie usuniesz konta. Znikające wiadomości są usuwane zgodnie z ustawionym przez ciebie timerem.",
        "Dziennik działań i zapisy logowań są przechowywane do jednego roku — są potrzebne do wyjaśniania włamań.",
        "Kopie zapasowe bazy danych są przechowywane w postaci zaszyfrowanej do 30 dni. Usunięte dane znikają z kopii w miarę ich wygasania.",
      ],
    },
    {
      title: "Usunięcie konta",
      body: [
        "Konto można usunąć w aplikacji: profil → „Usuń konto”, z potwierdzeniem hasłem. To samo jest dostępne na stronie usuwania konta, bez instalowania aplikacji.",
        "Usunięcie następuje od razu i nie można go cofnąć. Usuwane są: hasło i wszystkie sesje, klucze szyfrowania, tokeny push, twoje pliki i notatki, ustawienia, reakcje i głosy, treść twoich wiadomości prywatnych — także u rozmówcy.",
        "Pozostają: twoje wiadomości w czatach grupowych i posty w społecznościach (to cudza korespondencja i publiczne Aktualności) — już bez twojego imienia, podpisane jako „Usunięte konto”; dziennik bezpieczeństwa, którego nie można zmieniać wstecz.",
        "Grupy i społeczności, którymi zarządzasz, przechodzą na kolejną osobę zarządzającą; jeśli takiej nie ma — są usuwane razem z kontem. Twój login nie trafia do nikogo innego.",
      ],
    },
    {
      title: "Twoje prawa",
      body: [
        "Możesz otrzymać kopię swoich danych, poprawić je, usunąć konto lub sprzeciwić się przetwarzaniu — napisz do nas na adres podany na końcu dokumentu. Odpowiadamy w rozsądnym terminie, zwykle w ciągu 30 dni.",
        "Jeśli jesteś w UE lub Wielkiej Brytanii, masz prawo złożyć skargę do organu nadzorczego w swoim kraju.",
      ],
    },
    {
      title: "Bezpieczeństwo i jego granice",
      body: [
        "Co robimy: hasła są przechowywane wyłącznie jako hashe (Argon2id), prywatne rozmowy są szyfrowane na urządzeniach, ruch przechodzi przez TLS, działania administratorów są rejestrowane, kopie zapasowe są szyfrowane, działa ochrona przed odgadywaniem haseł i rejestracjami spamerskimi.",
        "Czego nie obiecujemy. Żaden serwis nie jest chroniony w sposób absolutny. Ryzyka, o których trzeba wiedzieć z góry:",
        "— Włamanie na serwer lub do infrastruktury dostawcy może ujawnić wszystko poza treścią prywatnych rozmów.",
        "— Dostęp do twojego urządzenia (kradzież, złośliwe oprogramowanie, obce ręce) ujawnia także prywatne rozmowy: klucze są przechowywane na urządzeniu.",
        "— Rozmówca może zapisać, przekazać dalej lub sfotografować to, co od ciebie dostanie. Technicznie nie da się temu zapobiec.",
        "— Utrata urządzenia lub ponowna instalacja aplikacji może oznaczać utratę historii prywatnych rozmów: klucze szyfrowania nie opuszczają urządzenia i nie możemy ich odtworzyć za ciebie.",
        "— Awaria hostingu, błąd w programie lub wyczerpanie bezpłatnego planu mogą doprowadzić do niedostępności serwisu i utraty danych. Kopie zapasowe ważnych rzeczy rób samodzielnie.",
        "— Powiadomienia push przechodzą przez Google i w drodze do urządzenia nie są przez nas szyfrowane; mogą się w nich znaleźć imię nadawcy i początek wiadomości, jeśli podgląd nie jest wyłączony.",
      ],
    },
    {
      title: "Dzieci",
      body: [
        "Serwis nie jest przeznaczony dla dzieci poniżej 13 lat (w UE — poniżej 16 lat lub wieku określonego w twoim kraju). Jeśli wiesz, że dziecko korzysta z serwisu bez zgody rodziców, napisz do nas, a usuniemy konto.",
      ],
    },
    {
      title: "Odpowiedzialność",
      body: [
        "Serwis jest udostępniany „tak jak jest” i „w miarę dostępności”, bez jakichkolwiek gwarancji: dostępności, zachowania danych, przydatności do określonego celu i braku błędów.",
        "W najszerszym zakresie dopuszczalnym przez obowiązujące prawo właściciel serwisu nie ponosi odpowiedzialności za: utratę lub uszkodzenie danych i korespondencji, niedostępność serwisu, brak możliwości odtworzenia kluczy szyfrowania i historii, treści tworzone przez użytkowników, działania innych użytkowników i osób trzecich, a także za szkody pośrednie, utracone korzyści i jakiekolwiek skutki korzystania lub niemożności korzystania z serwisu.",
        "Treść korespondencji tworzą użytkownicy. Właściciel serwisu nie sprawdza jej z góry i nie odpowiada za nią; w razie naruszenia zasad konto może zostać ograniczone lub usunięte.",
        "Powyższe zastrzeżenia nie wyłączają tego, czego zgodnie z prawem wyłączyć nie można: odpowiedzialności za działania umyślne i rażące niedbalstwo, za szkody na życiu i zdrowiu, a także praw konsumentów i obowiązków administratora danych osobowych w twoim kraju. Tam, gdzie takie ograniczenie jest niedopuszczalne, stosuje się je w minimalnym zakresie dopuszczalnym przez prawo.",
      ],
    },
    {
      title: "Zmiany",
      body: [
        "Możemy zmieniać tę politykę. Istotne zmiany są pokazywane w aplikacji. Data ostatniej zmiany jest podana na górze strony.",
      ],
    },
    {
      title: "Kontakt",
      body: [
        "Pytania dotyczące danych, wnioski o usunięcie i skargi: kisyandco@gmail.com. Odpowiadamy w rozsądnym terminie, zwykle w ciągu 30 dni.",
      ],
    },
  ],
  deletion: [
    {
      title: "Jak usunąć konto w aplikacji",
      body: [
        "1. Otwórz KISY i zaloguj się na swoje konto.",
        "2. Profil (ikona w prawym dolnym rogu) → „Usuń konto”.",
        "3. Wpisz hasło i słowo USUŃ.",
        "4. Gotowe: konto zostaje usunięte od razu, bez możliwości cofnięcia.",
      ],
    },
    {
      title: "Co zostaje usunięte",
      body: [
        "— Hasło, wszystkie aktywne sesje i urządzenia.",
        "— Klucze szyfrowania i tokeny push.",
        "— Twoje pliki, notatki, ustawienia, reakcje i głosy.",
        "— Treść twoich wiadomości prywatnych, także u rozmówcy.",
      ],
    },
    {
      title: "Co pozostaje",
      body: [
        "— Twoje wiadomości w czatach grupowych i posty w społecznościach — bez twojego imienia, podpisane jako „Usunięte konto”. To cudza korespondencja i publiczne Aktualności, a my nie możemy ich usuwać w imieniu innych.",
        "— Dziennik bezpieczeństwa (kto i kiedy się logował, działania administratorów) — do jednego roku, bez treści korespondencji.",
        "— Zapis o twojej akceptacji polityki i zasad społeczności: kiedy i w jakich wersjach. Bez niego nie da się udowodnić, że zgoda została wyrażona.",
        "— Zaszyfrowane kopie zapasowe bazy — do 30 dni, po czym znikają.",
      ],
    },
    {
      title: "Jeśli nie możesz się zalogować",
      body: [
        "Napisz na kisyandco@gmail.com, podając login, który chcesz usunąć. Usuniemy konto po sprawdzeniu, że należy do ciebie.",
      ],
    },
  ],
  rules: [
    {
      title: "W skrócie",
      body: [
        "KISY to miejsce do korespondencji, wspólnych grup i społeczności. Poniższe zasady obowiązują wszędzie, gdzie coś piszesz, przesyłasz lub pokazujesz innym: w czatach prywatnych i grupowych, w społecznościach i w Aktualnościach, w nazwach i opisach grup, w imieniu i awatarze profilu.",
        "Najważniejsza zasada: nie rób innym tego, za co w zwykłym życiu odpowiada się przed prawem lub przed ludźmi. Jeśli masz wątpliwości — nie publikuj.",
      ],
    },
    {
      title: "Co jest zabronione",
      body: [
        "— Wykorzystywanie seksualne dzieci i wszelkie materiały seksualizujące małoletnich. Tu nie ma ostrzeżeń: konto zostaje zablokowane natychmiast, a informacje są przekazywane organom ścigania.",
        "— Treści o charakterze jednoznacznie seksualnym i pornografia. Serwis jest przeznaczony dla użytkowników od 13 lat.",
        "— Groźby, nawoływanie do przemocy, gloryfikowanie przemocy i terroryzmu, werbowanie do organizacji ekstremistycznych.",
        "— Nękanie i prześladowanie: obelgi, systematyczne ataki na osobę, podburzanie innych, ponowne wiadomości do kogoś, kto cię zablokował lub poprosił o zaprzestanie kontaktu.",
        "— Podżeganie do nienawiści i poniżanie ludzi ze względu na narodowość, rasę, religię, płeć, orientację seksualną, niepełnosprawność, wiek lub pochodzenie.",
        "— Publikowanie cudzych danych osobowych bez zgody: adresu, telefonu, dokumentów, zdjęć z życia prywatnego, korespondencji.",
        "— Nakłanianie do samookaleczenia i samobójstwa, instrukcje, jak to zrobić, romantyzowanie zaburzeń odżywiania.",
        "— Spam i sztuczne nabijanie: masowe identyczne wiadomości, niezamówione reklamy, farmy kont, nabijanie reakcji i głosów.",
        "— Oszustwa i wprowadzanie w błąd: phishing, wyłudzanie pieniędzy i haseł, fałszywe konkursy, złośliwe pliki i linki.",
        "— Podszywanie się pod inną osobę lub organizację, w tym wprowadzające w błąd imię i awatar.",
        "— Sprzedaż i reklama rzeczy zakazanych: narkotyków, broni, fałszywych dokumentów, rzeczy kradzionych.",
        "— Naruszanie cudzych praw: publikowanie cudzych utworów, zdjęć i materiałów bez prawa do tego.",
        "— Obchodzenie ograniczeń: zakładanie nowego konta, aby kontynuować to, za co poprzednie zostało ograniczone lub usunięte.",
      ],
    },
    {
      title: "Grupy i społeczności",
      body: [
        "Osoba, która utworzyła grupę lub społeczność, oraz wyznaczeni przez nią redaktorzy odpowiadają za porządek wewnątrz i mogą usuwać posty. Wewnętrzne zasady społeczności nie mogą zezwalać na to, czego zabraniają niniejsze zasady.",
        "Zamknięta społeczność nie jest miejscem, w którym zasady nie obowiązują: zamknięcie chroni członków przed cudzymi oczami, a nie naruszenia przed moderacją.",
      ],
    },
    {
      title: "Rozmowy prywatne",
      body: [
        "Czaty prywatne są szyfrowane na urządzeniach i nie możemy ich odczytać — także w przypadku zgłoszenia: przy wiadomości z czatu prywatnego widzimy tylko sam fakt zgłoszenia. Dlatego główna ochrona jest tu w twoich rękach: rozmówcę możesz zablokować w dowolnym momencie, a decyzje dotyczące konta zapadają na podstawie wszystkich zgłoszeń łącznie.",
        "Szyfrowanie nie sprawia, że to, co zabronione, staje się dozwolone. Jeśli rozmówca sam pokaże naruszenie — na przykład zrzutem ekranu w wiadomości do nas — mamy prawo podjąć działania zgodnie z tymi zasadami.",
      ],
    },
    {
      title: "Jak się chronić i zgłaszać naruszenia",
      body: [
        "— Zablokuj. Przycisk w nagłówku czatu prywatnego z daną osobą. Blokada jest jednostronna i cicha: ta osoba się o niej nie dowie, nie będzie mogła pisać do ciebie w wiadomościach prywatnych ani dzwonić, a jej posty w społecznościach znikną z twoich Aktualności. Blokadę możesz zdjąć w profilu → „Zablokowani”.",
        "— Zgłosić można wiadomość (z jej menu), post w społeczności, osobę (w nagłówku czatu prywatnego lub na liście członków grupy) oraz samą społeczność lub grupę. Osoba, którą zgłaszasz, nie dowie się o tym.",
        "— Zgłaszać można tylko to, co widzisz na własne oczy.",
        "— Post w społeczności zgłoszony przez pięć różnych osób zostaje ukryty w Aktualnościach do czasu sprawdzenia. Zgłoszenia od kont utworzonych w ostatnich godzinach docierają do administracji, ale nie są wliczane do tych pięciu — inaczej post można by ukryć grupą świeżo założonych kont.",
        "— Zgłoszenia rozpatruje administracja serwisu. Jeśli ktoś ci grozi lub jesteś w niebezpieczeństwie, najpierw zwróć się do policji: nie jesteśmy służbą ratunkową.",
      ],
    },
    {
      title: "Konsekwencje naruszeń",
      body: [
        "W zależności od wagi i powtarzalności:",
        "— usunięcie posta w społeczności — przez jej redaktorów lub administrację;",
        "— dla grupy lub społeczności: ostrzeżenie, wykluczenie z ogólnych Aktualności, usunięcie (trzecie obowiązujące ostrzeżenie usuwa społeczność);",
        "— dla konta: blokada, po której nie można się na nie zalogować.",
        "Za poważne naruszenia — groźby wobec życia, wykorzystywanie seksualne dzieci, terroryzm — konto jest blokowane bez ostrzeżenia, a informacje mogą zostać przekazane organom ścigania w trybie przewidzianym przez prawo.",
        "Konto utworzone bez zaproszenia przez pierwsze godziny po rejestracji działa w trybie ograniczonym: to ochrona przed spamem, a nie kara.",
      ],
    },
    {
      title: "Jeśli nie zgadzasz się z decyzją",
      body: [
        "Napisz na kisyandco@gmail.com: podaj login, opisz, co się stało i dlaczego uważasz decyzję za błędną. Rozpatrzymy ją ponownie i odpowiemy, zwykle w ciągu 30 dni.",
      ],
    },
    {
      title: "Zmiany zasad",
      body: [
        "Zasady mogą się zmieniać. Przy istotnych zmianach aplikacja poprosi o zaakceptowanie nowej wersji przy następnym logowaniu — bez tego nie można korzystać z serwisu. Data ostatniej zmiany jest podana na górze strony.",
      ],
    },
  ],
};
