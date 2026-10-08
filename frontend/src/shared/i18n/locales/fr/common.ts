import type { Translation } from "../../types";
import type { common as source } from "../ru/common";

export const common: Translation<typeof source> = {
  "common.deletedAccount": "Compte supprimé",
  "common.today": "Aujourd'hui",
  "common.yesterday": "Hier",
  "common.justNow": "à l'instant",
  "common.minutesAgo": "il y a {count} min",
  "common.hoursAgo": "il y a {count} h",
  "common.rateLimited": "Trop de tentatives. Réessayez un peu plus tard",
  "common.rateLimitedSeconds": "Trop de tentatives. Réessayez dans {count} s",
  "common.rateLimitedMinutes": "Trop de tentatives. Réessayez dans {count} min",

  "common.loading": "Chargement",
  "common.modal.close": "Fermer",
  "common.code.copy": "Copier",

  "common.verified.account": "Compte vérifié",
  "common.verified.community": "Communauté vérifiée",

  "common.media.resetZoom": "Réinitialiser le zoom (0)",
  "common.media.download": "Télécharger",
  "common.media.close": "Fermer (Échap)",
  "common.media.previous": "Précédent",
  "common.media.next": "Suivant",

  "common.emoji.picker": "Sélecteur d'emoji",
  "common.emoji.search": "Rechercher un emoji",
  "common.emoji.noResults": "Aucun résultat",
  "common.emoji.recent": "Récents",
  "common.emoji.smileys": "Smileys",
  "common.emoji.gestures": "Gestes",
  "common.emoji.hearts": "Cœurs",
  "common.emoji.objects": "Objets",

  "common.nav.messages": "Messages",
  "common.nav.communities": "Communautés",
  "common.nav.rating": "Classement",
  "common.nav.feed": "Fil",

  "common.push.channelName": "Messages",
  "common.push.channelDescription": "Nouveaux messages et mentions",

  "common.displayName.taken": "Ce nom est déjà pris",
  "common.displayName.letters": "Uniquement des lettres, avec un seul espace entre les mots",
  "common.displayName.length": "Nom : de 2 à 40 caractères",

  "common.password.rule": "12 à 128 caractères, dont au moins une lettre et un chiffre",
  "common.password.tooShort": "Au moins {min} caractères",
  "common.password.tooLong": "Pas plus de {max} caractères",
  "common.password.needLetter": "Ajoutez au moins une lettre",
  "common.password.needDigit": "Ajoutez au moins un chiffre",

  "common.quarantine.inAnHour": "dans une heure",
  "common.quarantine.inHours": {
    one: "dans {count} heure",
    many: "dans {count} heures",
    other: "dans {count} heures",
  },
  "common.quarantine.heldBack": "{feature} {when}",
  "common.quarantine.fileTooLarge":
    "Un nouveau compte peut envoyer des fichiers jusqu'à {size}. Cette limite sera levée {when}",

  "common.units.bytes": "{value} o",
  "common.units.kb": "{value} Ko",
  "common.units.mb": "{value} Mo",
  "common.units.gb": "{value} Go",

  "common.duration.oneDay": "1 jour",
  "common.duration.days": "{count} j",
  "common.duration.oneHour": "1 heure",
  "common.duration.hours": "{count} h",
  "common.duration.minutes": "{count} min",
  "common.duration.seconds": "{count} s",

  "common.group.group": "Groupe",
  "common.group.community": "Communauté",
  "common.group.openCommunity": "Communauté publique",
  "common.group.fromRole": "{kind} · {role} et au-dessus",

  "common.session.refreshFailed": "Impossible de renouveler la session",
  "common.consent.saveFailed": "Impossible d'enregistrer votre consentement. Réessayez",

  "common.offline.title": "Pas de connexion au serveur",
  "common.offline.body":
    "Votre session reste ouverte — l'application se reconnectera d'elle-même dès que le réseau sera de retour.",
  "common.offline.retry": "Réessayer",

  "common.e2ee.deviceNotInChat":
    "Cet appareil n'a pas encore rejoint la discussion chiffrée. Votre interlocuteur ou un autre de vos appareils l'ajoutera dès qu'il sera en ligne — renvoyez alors le message.",
  "common.e2ee.peerNoDevices":
    "Votre interlocuteur ne s'est pas encore connecté à une version de KISY prenant en charge le chiffrement — le message n'a pas été envoyé.",
  "common.e2ee.peerKeysExhausted":
    "Votre interlocuteur n'a plus de clés de chiffrement — le message n'a pas été envoyé. Demandez-lui d'ouvrir KISY, puis réessayez.",
  "common.e2ee.unavailable":
    "Le chiffrement n'a pas pu démarrer sur cet appareil — le message n'a pas été envoyé. Redémarrez l'application et réessayez.",
  "common.e2ee.peerUnknown": "Impossible d'identifier votre interlocuteur — le message n'a pas été envoyé.",
  "common.e2ee.encryptFailed": "Impossible de chiffrer le message — il n'a pas été envoyé. Réessayez.",

  "common.forward.nothingToForward": "Aucun de ces messages ne peut être transféré",
};
