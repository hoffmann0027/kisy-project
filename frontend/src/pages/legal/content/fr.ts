// Traduction française de content/ru.ts. Seule la version russe fait foi ; ceci est une traduction de courtoisie.
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-08",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "En bref",
      body: [
        "KISY est une messagerie. Pour qu'elle fonctionne, nous conservons votre compte, vos conversations et les fichiers que vous envoyez. Nous ne vendons pas ces données, n'affichons pas de publicité et ne les transmettons pas à des tiers, sauf dans les cas décrits ci-dessous.",
        "Les conversations privées en tête-à-tête sont chiffrées sur les appareils : le serveur les conserve sous une forme qu'il ne peut pas lire lui-même. Les messages dans les groupes et les publications dans les communautés sont conservés en clair — ils sont visibles pour l'administrateur du serveur.",
        "Vous pouvez supprimer votre compte à tout moment : Profil → « Supprimer le compte », ou sur la page de suppression du compte, sans vous connecter à l'application.",
      ],
    },
    {
      title: "Qui est responsable des données",
      body: [
        "Le service fonctionne comme un projet privé. Le responsable du traitement des données est le propriétaire de l'installation KISY sur laquelle vous avez créé votre compte — c'est aussi lui qui répond aux questions sur vos données, à l'adresse indiquée à la fin du document.",
        "L'application est gratuite et distribuée « en l'état », sans garantie de disponibilité ni de conservation des données (voir la section « Responsabilité »).",
      ],
    },
    {
      title: "Quelles données nous collectons",
      body: [
        "— Compte : identifiant, nom affiché, hachage du mot de passe (le mot de passe lui-même n'est pas conservé), date d'inscription, niveau d'accès, avatar si vous en avez importé un.",
        "— Contenu : messages, fichiers, messages vocaux, notes, publications dans les communautés, réactions, calendrier et tâches — tout ce que vous créez dans l'application.",
        "— Données techniques : heures de connexion, empreinte irréversible (hachage) de l'adresse IP, nom de l'appareil et du navigateur, identifiants des notifications push, journal des actions des administrateurs.",
        "— Appels : le fait qu'un appel a eu lieu, son heure et sa durée. Le contenu des appels n'est pas enregistré.",
        "— Consentement : quand vous avez accepté cette politique et les règles de la communauté, dans quelles versions, et l'empreinte irréversible de votre adresse IP à ce moment-là. C'est la preuve de votre consentement : elle est donc conservée même après la suppression du compte — tout comme le journal de sécurité.",
        "Nous ne demandons pas et ne conservons pas votre numéro de téléphone, votre adresse, vos données de paiement ni votre géolocalisation précise.",
      ],
    },
    {
      title: "Pourquoi nous en avons besoin",
      body: [
        "— Pour acheminer les messages et afficher les conversations sur vos appareils.",
        "— Pour protéger le service : limiter le spam et les tentatives de deviner les mots de passe, détecter les abus (c'est pour cela qu'il nous faut le hachage de l'adresse IP, et non l'adresse elle-même).",
        "— Pour vous envoyer des notifications, si vous les avez activées.",
        "— Pour respecter les exigences de la loi, lorsqu'elles nous sont applicables.",
        "Nous n'utilisons pas vos données pour la publicité, le profilage ni l'entraînement de modèles.",
      ],
    },
    {
      title: "Qui a accès aux données",
      body: [
        "— Les autres utilisateurs — exactement les personnes à qui vous écrivez, et les membres des groupes et communautés où vous écrivez.",
        "— Les fournisseurs d'infrastructure sans lesquels le service ne fonctionne pas : hébergement de l'application et de la base de données, stockage des fichiers, service d'acheminement des notifications push (Google Firebase), protection contre les bots (Cloudflare Turnstile). Ils traitent les données pour notre compte et ne les utilisent pas à leurs propres fins.",
        "— L'administrateur du serveur — dans la mesure décrite à la section « Ce que voit l'administrateur ».",
        "Nous ne vendons pas les données et ne les transmettons pas aux réseaux publicitaires.",
      ],
    },
    {
      title: "Ce que voit l'administrateur",
      body: [
        "En toute franchise, voici les limites du chiffrement. L'administrateur du serveur ne peut pas lire les conversations privées en tête-à-tête : elles sont chiffrées avec des clés qui restent sur les appareils.",
        "L'administrateur peut voir : les messages des discussions de groupe et les publications des communautés, les noms et les identifiants, le fait et l'heure des échanges (qui écrit à qui et quand), les fichiers importés, le journal des actions. Lorsqu'un message privé est signalé, nous ne voyons que l'existence du signalement — le texte reste chiffré.",
        "Si vous avez besoin d'une confidentialité totale de vos échanges, utilisez les discussions privées, pas les groupes.",
      ],
    },
    {
      title: "Durée de conservation",
      body: [
        "Les messages et les fichiers sont conservés jusqu'à ce que vous les supprimiez ou que vous supprimiez votre compte. Les messages éphémères sont supprimés selon le minuteur que vous avez défini.",
        "Le journal des actions et les enregistrements de connexion sont conservés jusqu'à un an — ils sont nécessaires pour enquêter sur les piratages.",
        "Les sauvegardes de la base de données sont conservées sous forme chiffrée jusqu'à 30 jours. Les données supprimées disparaissent des sauvegardes à mesure que celles-ci expirent.",
      ],
    },
    {
      title: "Suppression du compte",
      body: [
        "Vous pouvez supprimer votre compte dans l'application : Profil → « Supprimer le compte », avec confirmation par mot de passe. La même chose est possible sur la page de suppression du compte, sans installer l'application.",
        "La suppression est immédiate et ne peut pas être annulée. Sont supprimés : le mot de passe et toutes les sessions, les clés de chiffrement, les jetons push, vos fichiers et notes, les paramètres, les réactions et les votes, les textes de vos messages privés — y compris chez votre interlocuteur.",
        "Restent : vos messages dans les discussions de groupe et vos publications dans les communautés (ce sont les échanges d'autres personnes et un fil d'actualité public) — désormais sans votre nom, sous « Compte supprimé » ; le journal de sécurité, qui ne peut pas être réécrit a posteriori.",
        "Les groupes et communautés que vous gériez passent au responsable suivant ; s'il n'y en a pas, ils sont supprimés avec le compte. Votre identifiant n'est attribué à personne d'autre.",
      ],
    },
    {
      title: "Vos droits",
      body: [
        "Vous pouvez obtenir une copie de vos données, les rectifier, supprimer votre compte ou vous opposer au traitement — écrivez-nous à l'adresse indiquée à la fin du document. Nous répondons dans un délai raisonnable, généralement sous 30 jours.",
        "Si vous vous trouvez dans l'UE ou au Royaume-Uni, vous avez le droit d'introduire une réclamation auprès de l'autorité de contrôle de votre pays.",
      ],
    },
    {
      title: "La sécurité et ses limites",
      body: [
        "Ce que nous faisons : les mots de passe ne sont conservés que sous forme de hachages (Argon2id), les conversations privées sont chiffrées sur les appareils, le trafic passe par TLS, les actions des administrateurs sont journalisées, les sauvegardes sont chiffrées, et une protection existe contre les tentatives de deviner les mots de passe et contre les inscriptions de spam.",
        "Ce que nous ne promettons pas. Aucun service n'est protégé de façon absolue. Voici les risques que vous devez connaître à l'avance :",
        "— Un piratage du serveur ou de l'infrastructure d'un fournisseur peut exposer tout, sauf le contenu des conversations privées.",
        "— Un accès à votre appareil (vol, logiciel malveillant, quelqu'un d'autre qui l'utilise) expose aussi les conversations privées : les clés sont conservées sur l'appareil.",
        "— Votre interlocuteur peut enregistrer, transférer ou photographier ce que vous lui avez envoyé. Techniquement, cela ne peut pas être empêché.",
        "— La perte de l'appareil ou la réinstallation de l'application peut entraîner la perte de l'historique des conversations privées : les clés de chiffrement ne quittent pas l'appareil, et nous ne pouvons pas les récupérer à votre place.",
        "— Une panne de l'hébergement, une erreur dans le logiciel ou l'épuisement de l'offre gratuite peuvent rendre le service indisponible et entraîner une perte de données. Sauvegardez vous-même ce qui est important.",
        "— Les notifications push passent par Google et ne sont pas chiffrées par nous sur leur trajet vers l'appareil ; elles peuvent contenir le nom de l'expéditeur et le début du message, si vous n'avez pas désactivé l'aperçu.",
      ],
    },
    {
      title: "Enfants",
      body: [
        "Le service n'est pas destiné aux enfants de moins de 13 ans (dans l'UE — de moins de 16 ans ou de l'âge fixé par votre pays). Si vous apprenez qu'un enfant utilise le service sans le consentement de ses parents, écrivez-nous et nous supprimerons le compte.",
      ],
    },
    {
      title: "Responsabilité",
      body: [
        "Le service est gratuit et fourni « en l'état » et « selon disponibilité », sans aucune garantie, expresse ou implicite : de disponibilité, de fonctionnement ininterrompu, de conservation des données, d'adéquation à un usage particulier ni d'absence d'erreurs et de vulnérabilités. Vous l'utilisez à vos risques et périls.",
        "Le propriétaire du service n'est pas tenu de conserver, de sauvegarder, de restaurer ou de remettre vos données et vos conversations, ni de vérifier et de modérer le contenu, ni de maintenir le fonctionnement du service ou de certaines de ses fonctionnalités. Le service, comme chacune de ses fonctionnalités, peut être modifié, suspendu ou arrêté à tout moment, sans préavis.",
        "Dans toute la mesure permise par la loi applicable, le propriétaire du service n'est pas responsable : de la perte ou de l'altération des données et des conversations, de l'indisponibilité du service, de l'impossibilité de récupérer les clés de chiffrement et l'historique, du contenu créé par les utilisateurs, des actions d'autres utilisateurs, de tiers et des services tiers sur lesquels KISY fonctionne, ni des dommages directs et indirects, du manque à gagner et de toute conséquence de l'utilisation ou de l'impossibilité d'utiliser le service.",
        "Si la responsabilité du propriétaire du service est néanmoins engagée, son montant total est limité à la somme que vous avez payée pour le service au cours des 12 derniers mois. Le service étant gratuit, cette somme est égale à zéro.",
        "Vous assumez l'entière responsabilité de la manière dont vous utilisez KISY : du contenu que vous créez, envoyez et publiez, du respect des lois et des droits d'autrui, de la protection de votre mot de passe, de vos appareils et de vos clés de chiffrement. Si des réclamations sont formulées contre le propriétaire du service en raison de votre contenu ou de vos actions, vous vous engagez à l'indemniser de ses pertes et de ses frais.",
        "Le contenu des conversations est créé par les utilisateurs. Le propriétaire du service ne le vérifie pas à l'avance et n'en est pas responsable ; en cas de violation des règles, le compte peut être restreint ou supprimé.",
        "Les réserves ci-dessus n'écartent pas ce dont la loi interdit de s'exonérer : la responsabilité pour les actes intentionnels et la faute lourde, pour les atteintes à la vie et à la santé, ainsi que les droits des consommateurs et les obligations du responsable du traitement des données personnelles dans votre pays. Là où la limitation de responsabilité n'est pas admise en totalité, elle s'applique dans la mesure maximale permise par la loi.",
      ],
    },
    {
      title: "Modifications",
      body: [
        "Nous pouvons modifier cette politique. Les modifications importantes sont signalées dans l'application. La date de la dernière modification est indiquée en haut de la page.",
      ],
    },
    {
      title: "Contact",
      body: [
        "Questions sur les données, demandes de suppression et réclamations : kisyandco@gmail.com. Nous répondons dans un délai raisonnable, généralement sous 30 jours.",
      ],
    },
  ],
  deletion: [
    {
      title: "Comment supprimer votre compte depuis l'application",
      body: [
        "1. Ouvrez KISY et connectez-vous à votre compte.",
        "2. Profil (l'icône en bas à droite) → « Supprimer le compte ».",
        "3. Saisissez votre mot de passe et le mot SUPPRIMER.",
        "4. C'est fait : le compte est supprimé immédiatement, sans possibilité d'annulation.",
      ],
    },
    {
      title: "Ce qui est supprimé",
      body: [
        "— Le mot de passe, toutes les sessions actives et tous les appareils.",
        "— Les clés de chiffrement et les jetons push.",
        "— Vos fichiers, notes, paramètres, réactions et votes.",
        "— Les textes de vos messages privés, y compris chez votre interlocuteur.",
      ],
    },
    {
      title: "Ce qui reste",
      body: [
        "— Vos messages dans les discussions de groupe et vos publications dans les communautés — sans votre nom, sous « Compte supprimé ». Ce sont les échanges d'autres personnes et un fil d'actualité public, et nous ne pouvons pas les effacer à la place des autres.",
        "— Le journal de sécurité (qui s'est connecté et quand, actions des administrateurs) — jusqu'à un an, sans le contenu des conversations.",
        "— L'enregistrement attestant que vous avez accepté la politique de confidentialité et les règles de la communauté : quand, et dans quelles versions. Sans lui, il est impossible de prouver que le consentement a été donné.",
        "— Les sauvegardes chiffrées de la base — jusqu'à 30 jours, après quoi elles disparaissent.",
      ],
    },
    {
      title: "Si vous ne parvenez pas à vous connecter",
      body: [
        "Écrivez à kisyandco@gmail.com en indiquant l'identifiant que vous souhaitez supprimer. Nous supprimerons le compte après avoir vérifié qu'il vous appartient.",
      ],
    },
  ],
  rules: [
    {
      title: "En bref",
      body: [
        "KISY est un lieu d'échanges, de groupes partagés et de communautés. Les règles ci-dessous s'appliquent partout où vous écrivez, importez ou montrez quelque chose à d'autres : dans les discussions privées et de groupe, dans les communautés et leur fil d'actualité, dans les noms et descriptions des groupes, dans le nom et l'avatar de votre profil.",
        "Règle principale : ne faites pas aux autres ce dont, dans la vie courante, on doit répondre devant la loi ou devant les autres. Dans le doute, ne publiez pas.",
        "Vous répondez vous-même de tout ce que vous écrivez, envoyez et publiez, ainsi que des conséquences qui en découlent. KISY ne vérifie pas le contenu à l'avance et n'en est pas responsable.",
      ],
    },
    {
      title: "Ce qui est interdit",
      body: [
        "— Les abus sexuels sur des enfants et tout contenu sexualisant des mineurs. Ici, pas d'avertissement : le compte est bloqué immédiatement et les informations sont transmises aux forces de l'ordre.",
        "— Les contenus sexuels explicites et la pornographie. Le service s'adresse aux utilisateurs de 13 ans et plus.",
        "— Les menaces, les appels à la violence, l'apologie de la violence et du terrorisme, le recrutement pour des organisations extrémistes.",
        "— Le harcèlement et la persécution : insultes, attaques systématiques contre une personne, fait de monter d'autres personnes contre quelqu'un, messages répétés à quelqu'un qui vous a bloqué ou vous a demandé d'arrêter.",
        "— L'incitation à la haine et l'humiliation de personnes en raison de leur nationalité, de leur race, de leur religion, de leur sexe, de leur orientation sexuelle, de leur handicap, de leur âge ou de leur origine.",
        "— La publication des données personnelles d'autrui sans son consentement : adresse, téléphone, documents, photos de sa vie privée, correspondance.",
        "— Les incitations à l'automutilation et au suicide, les instructions pour y parvenir, la romantisation des troubles du comportement alimentaire.",
        "— Le spam et la manipulation de l'audience : messages identiques en masse, publicité non sollicitée, fermes de comptes, gonflage artificiel des réactions et des votes.",
        "— La fraude et la tromperie : hameçonnage, soutirage d'argent et de mots de passe, faux concours, fichiers et liens malveillants.",
        "— Le fait de se faire passer pour une autre personne ou une organisation, y compris au moyen d'un nom et d'un avatar trompeurs.",
        "— La vente et la promotion de produits interdits : drogues, armes, faux documents, biens volés.",
        "— L'atteinte aux droits d'autrui : publication d'œuvres, de photos et de contenus d'autrui sans en avoir le droit.",
        "— Le contournement des restrictions : créer un nouveau compte pour continuer ce qui a valu au précédent d'être restreint ou supprimé.",
      ],
    },
    {
      title: "Groupes et communautés",
      body: [
        "La personne qui a créé un groupe ou une communauté, et les éditeurs qu'elle a nommés, sont responsables de l'ordre à l'intérieur et peuvent supprimer des publications. Les règles internes d'une communauté ne peuvent pas autoriser ce que les présentes règles interdisent.",
        "Une communauté privée n'est pas un lieu où les règles ne s'appliquent pas : le caractère privé protège les membres des regards extérieurs, pas les infractions de la modération.",
      ],
    },
    {
      title: "Conversations privées",
      body: [
        "Les discussions privées sont chiffrées sur les appareils, et nous ne pouvons pas les lire — y compris en cas de signalement : pour un message d'une discussion privée, nous ne voyons que l'existence du signalement. C'est pourquoi la principale protection, ici, est entre vos mains : vous pouvez bloquer votre interlocuteur à tout moment, et les décisions concernant un compte sont prises au vu de l'ensemble des signalements.",
        "Le chiffrement ne rend pas autorisé ce qui est interdit. Si votre interlocuteur nous montre lui-même une infraction — par exemple au moyen d'une capture d'écran jointe à sa demande —, nous sommes en droit de prendre des mesures en vertu des présentes règles.",
      ],
    },
    {
      title: "Comment vous protéger et signaler",
      body: [
        "— Bloquer. Le bouton se trouve dans l'en-tête de la discussion privée avec la personne. Le blocage est unilatéral et discret : la personne n'en saura rien, ne pourra plus vous écrire en privé ni vous appeler, et ses publications dans les communautés disparaîtront de votre fil d'actualité. Vous pouvez débloquer depuis Profil → « Bloqués ».",
        "— Vous pouvez signaler un message (depuis son menu), une publication dans une communauté, une personne (dans l'en-tête d'une discussion privée ou dans la liste des membres d'un groupe) ainsi que la communauté ou le groupe lui-même. La personne que vous signalez n'en saura rien.",
        "— Vous ne pouvez signaler que ce que vous voyez vous-même.",
        "— Une publication dans une communauté signalée par cinq personnes différentes est masquée du fil d'actualité jusqu'à ce qu'elle soit examinée. Les signalements provenant de comptes créés au cours des dernières heures parviennent bien à l'administration, mais ne comptent pas parmi ces cinq — sinon, une publication pourrait être masquée par une poignée de comptes tout neufs.",
        "— Les signalements sont examinés par l'administration du service. Si l'on vous menace ou si vous êtes en danger, adressez-vous d'abord à la police : nous ne sommes pas un service d'urgence.",
      ],
    },
    {
      title: "Sanctions en cas d'infraction",
      body: [
        "Selon la gravité et la récidive :",
        "— suppression d'une publication dans une communauté — par ses éditeurs ou par l'administration ;",
        "— pour un groupe ou une communauté : avertissement, exclusion du fil d'actualité général, suppression (le troisième avertissement en vigueur supprime la communauté) ;",
        "— pour un compte : blocage, après lequel il n'est plus possible de s'y connecter.",
        "Pour les infractions graves — menaces contre la vie, abus sexuels sur des enfants, terrorisme —, le compte est bloqué sans avertissement, et les informations peuvent être transmises aux forces de l'ordre selon la procédure prévue par la loi.",
        "Un compte créé sans invitation fonctionne en mode restreint pendant les premières heures suivant l'inscription : c'est une protection contre le spam, pas une sanction.",
      ],
    },
    {
      title: "Si vous contestez une décision",
      body: [
        "Écrivez à kisyandco@gmail.com : votre identifiant, ce qui s'est passé et pourquoi vous estimez la décision erronée. Nous la réexaminerons et vous répondrons, généralement sous 30 jours.",
      ],
    },
    {
      title: "Modification des règles",
      body: [
        "Les règles peuvent changer. En cas de modification importante, l'application vous demandera d'accepter la nouvelle version lors de votre prochaine connexion — il n'est pas possible d'utiliser le service sans cela. La date de la dernière modification est indiquée en haut de la page.",
      ],
    },
  ],
};
