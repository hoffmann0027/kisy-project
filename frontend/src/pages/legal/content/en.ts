// English translation of content/ru.ts. The Russian text is the legally binding one; this is a convenience translation (the app says so above it).
import type { LegalDocs } from "../legalContent";

export const docs: LegalDocs = {
  privacyUpdated: "2026-10-07",
  rulesUpdated: "2026-10-08",
  privacy: [
    {
      title: "In short",
      body: [
        "KISY is a messenger. To make it work, we store your account, your conversations and the files you send. We do not sell this data, do not show ads and do not pass it on to third parties, except in the cases described below.",
        "Private one-to-one conversations are encrypted on the devices: the server stores them in a form it cannot read itself. Messages in groups and posts in communities are stored unencrypted — the server administrator can see them.",
        "You can delete your account at any time: Profile → “Delete account”, or on the account deletion page without signing in to the app.",
      ],
    },
    {
      title: "Who is responsible for the data",
      body: [
        "The service is run as a private project. The data controller is the owner of the KISY installation you signed up on — they are also the one who answers questions about your data at the address given at the end of this document.",
        "The app is free and is distributed “as is”, without guarantees of availability or data preservation (see the “Liability” section).",
      ],
    },
    {
      title: "What data we collect",
      body: [
        "— Account: username, display name, password hash (the password itself is not stored), sign-up date, access level, and your avatar if you uploaded one.",
        "— Content: messages, files, voice messages, notes, community posts, reactions, calendar and tasks — everything you create in the app.",
        "— Technical: sign-in times, an irreversible fingerprint (hash) of your IP address, device and browser name, push notification identifiers, the log of administrators' actions.",
        "— Calls: the fact, time and duration of a call. The content of calls is not recorded.",
        "— Consent: when you accepted this policy and the Community Guidelines, which versions of them, and an irreversible fingerprint of your IP address at that moment. This is proof of consent, so it is kept even after your account is deleted — just like the security log.",
        "We do not ask for or store your phone number, address, payment details or precise location.",
      ],
    },
    {
      title: "Why we need it",
      body: [
        "— To deliver messages and show your conversations on your devices.",
        "— To protect the service: to limit spam and password guessing and to catch abuse (this is why we need a hash of the IP address rather than the address itself).",
        "— To send you notifications, if you have turned them on.",
        "— To meet legal requirements where they apply to us.",
        "We do not use your data for advertising, profiling or model training.",
      ],
    },
    {
      title: "Who can access the data",
      body: [
        "— Other users — exactly the people you write to, and the members of the groups and communities where you post.",
        "— Infrastructure providers without which the service cannot work: app and database hosting, file storage, the push notification delivery service (Google Firebase), bot protection (Cloudflare Turnstile). They process data on our behalf and do not use it for their own purposes.",
        "— The server administrator — to the extent described in the “What the administrator can see” section.",
        "We do not sell data or pass it on to advertising networks.",
      ],
    },
    {
      title: "What the administrator can see",
      body: [
        "An honest word about the limits of encryption. The server administrator cannot read private one-to-one conversations: they are encrypted with keys that stay on the devices.",
        "The administrator can see: messages in group chats and posts in communities, names and usernames, the fact and time of conversations (who talked to whom and when), uploaded files, the action log. When a private message is reported, we see only the fact of the report — the text stays encrypted.",
        "If you need your conversations to be completely private, use private chats, not groups.",
      ],
    },
    {
      title: "How long we keep data",
      body: [
        "Messages and files are kept until you delete them or delete your account. Disappearing messages are deleted according to the timer you set.",
        "The action log and sign-in records are kept for up to one year — they are needed to investigate security breaches.",
        "Database backups are kept in encrypted form for up to 30 days. Deleted data disappears from the backups as they expire.",
      ],
    },
    {
      title: "Account deletion",
      body: [
        "You can delete your account in the app: Profile → “Delete account”, confirming with your password. The same is available on the account deletion page without installing the app.",
        "Deletion happens immediately and cannot be undone. What is deleted: your password and all sessions, encryption keys, push tokens, your files and notes, settings, reactions and votes, and the texts of your private messages — including on the other person's side.",
        "What remains: your messages in group chats and posts in communities (these are other people's conversations and a public feed) — no longer under your name, shown as from “Deleted account”; and the security log, which cannot be rewritten after the fact.",
        "Groups and communities you managed pass to the next person in charge; if there is no such person, they are deleted together with the account. Your username will not go to anyone else.",
      ],
    },
    {
      title: "Your rights",
      body: [
        "You can get a copy of your data, correct it, delete your account or object to the processing — write to us at the address at the end of this document. We reply within a reasonable time, usually within 30 days.",
        "If you are in the EU or the UK, you have the right to lodge a complaint with the supervisory authority of your country.",
      ],
    },
    {
      title: "Security and its limits",
      body: [
        "What we do: passwords are stored only as hashes (Argon2id), private conversations are encrypted on the devices, traffic goes over TLS, administrators' actions are logged, backups are encrypted, and there is protection against password guessing and spam sign-ups.",
        "What we do not promise. No service is completely secure. Risks you should know about in advance:",
        "— A breach of the server or of a provider's infrastructure may expose everything except the content of private conversations.",
        "— Access to your device (theft, malware, someone else getting hold of it) exposes your private conversations too: the keys are stored on the device.",
        "— The person you are talking to can save, forward or take a photo of what you sent them. This cannot be prevented technically.",
        "— Losing your device or reinstalling the app may mean losing the history of your private conversations: the encryption keys never leave the device, and we cannot recover them for you.",
        "— A hosting failure, a bug in the software or the free plan running out may make the service unavailable and lead to data loss. Back up anything important yourself.",
        "— Push notifications pass through Google and are not encrypted by us on their way to the device; they may contain the sender's name and the beginning of the message unless you have turned off previews.",
      ],
    },
    {
      title: "Children",
      body: [
        "The service is not intended for children under 13 (in the EU — under 16 or the age set by your country). If you find out that a child is using the service without their parents' consent, write to us and we will delete the account.",
      ],
    },
    {
      title: "Liability",
      body: [
        "The service is provided “as is” and “as available”, without any warranties: of availability, data preservation, fitness for a particular purpose or freedom from errors.",
        "To the maximum extent permitted by applicable law, the owner of the service is not liable for: loss of or damage to data and conversations, unavailability of the service, inability to recover encryption keys and history, content created by users, actions of other users and third parties, or for indirect losses, lost profits and any consequences of using or being unable to use the service.",
        "The content of conversations is created by users. The owner of the service does not check it in advance and is not responsible for it; if the rules are broken, the account may be restricted or deleted.",
        "The disclaimers above do not exclude what cannot be excluded by law: liability for intentional acts and gross negligence, for harm to life and health, or consumer rights and the obligations of a personal data controller in your country. Where such a limitation is not permitted, it applies to the minimum extent permitted by law.",
      ],
    },
    {
      title: "Changes",
      body: [
        "We may change this policy. Significant changes are shown in the app. The date of the last change is given at the top of the page.",
      ],
    },
    {
      title: "Contact",
      body: [
        "Questions about data, deletion requests and complaints: kisyandco@gmail.com. We reply within a reasonable time, usually within 30 days.",
      ],
    },
  ],
  deletion: [
    {
      title: "How to delete your account in the app",
      body: [
        "1. Open KISY and sign in to your account.",
        "2. Profile (the icon at the bottom right) → “Delete account”.",
        "3. Enter your password and the word DELETE.",
        "4. Done: the account is deleted immediately, and this cannot be undone.",
      ],
    },
    {
      title: "What is deleted",
      body: [
        "— Your password, all active sessions and devices.",
        "— Encryption keys and push tokens.",
        "— Your files, notes, settings, reactions and votes.",
        "— The texts of your private messages, including on the other person's side.",
      ],
    },
    {
      title: "What remains",
      body: [
        "— Your messages in group chats and posts in communities — without your name, shown as from “Deleted account”. These are other people's conversations and a public feed, and we cannot erase them on behalf of others.",
        "— The security log (who signed in and when, administrators' actions) — for up to one year, without the content of conversations.",
        "— A record that you accepted the Privacy Policy and the Community Guidelines: when, and which versions of them. Without it, there is no way to prove that consent was given.",
        "— Encrypted database backups — for up to 30 days, after which they disappear.",
      ],
    },
    {
      title: "If you can't sign in",
      body: [
        "Write to kisyandco@gmail.com with the username you want to delete. We will delete the account after checking that it is yours.",
      ],
    },
  ],
  rules: [
    {
      title: "In short",
      body: [
        "KISY is a place for conversations, shared groups and communities. The guidelines below apply wherever you write, upload or show something to others: in private and group chats, in communities and their feed, in the names and descriptions of groups, and in your profile name and avatar.",
        "The main rule: don't do to others anything that, in everyday life, you would have to answer for before the law or before other people. If in doubt, don't post it.",
      ],
    },
    {
      title: "What is prohibited",
      body: [
        "— Child sexual abuse and any material that sexualizes minors. There are no warnings here: the account is blocked immediately, and the information is passed on to law enforcement.",
        "— Sexually explicit content and pornography. The service is intended for users aged 13 and over.",
        "— Threats, calls for violence, glorification of violence and terrorism, recruitment into extremist organizations.",
        "— Bullying and harassment: insults, systematic attacks on a person, setting others on someone, repeated messages to someone who has blocked you or asked you to stop.",
        "— Inciting hatred and degrading people on the basis of nationality, race, religion, gender, sexual orientation, disability, age or origin.",
        "— Publishing other people's personal data without their consent: addresses, phone numbers, documents, photos of their private life, correspondence.",
        "— Encouraging self-harm and suicide, instructions for them, romanticizing eating disorders.",
        "— Spam and fake engagement: mass identical messages, unsolicited advertising, account farms, artificially inflating reactions and votes.",
        "— Fraud and deception: phishing, tricking people out of money and passwords, fake giveaways, malicious files and links.",
        "— Impersonating another person or organization, including a misleading name and avatar.",
        "— Selling and advertising prohibited goods: drugs, weapons, forged documents, stolen property.",
        "— Infringing the rights of others: publishing other people's works, photos and materials without the right to do so.",
        "— Evading restrictions: creating a new account to carry on with what your previous account was restricted or deleted for.",
      ],
    },
    {
      title: "Groups and communities",
      body: [
        "Whoever created a group or community, and the editors they appoint, are responsible for keeping order inside it and can delete posts. A community's own rules cannot allow what these guidelines prohibit.",
        "A private community is not a place where the guidelines don't apply: being private protects the members from outsiders' eyes, not violations from moderation.",
      ],
    },
    {
      title: "Private conversations",
      body: [
        "Private chats are encrypted on the devices, and we cannot read them — not even when something is reported: for a message from a private chat, we see only the fact that it was reported. That is why the main protection here is in your hands: you can block the other person at any time, and decisions about an account are made on the basis of all the reports taken together.",
        "Encryption does not make what is prohibited allowed. If the other person shows us a violation themselves — for example, with a screenshot when contacting us — we are entitled to take action under these guidelines.",
      ],
    },
    {
      title: "How to protect yourself and report",
      body: [
        "— Block. The button is in the header of your private chat with the person. Blocking is one-way and silent: they won't find out about it, won't be able to send you private messages or call you, and their posts in communities will disappear from your feed. You can unblock them in Profile → “Blocked”.",
        "— You can report a message (from its menu), a community post, a person (in the header of a private chat or in a group's member list) and the community or group itself. The one you report won't find out about it.",
        "— You can only report what you can see yourself.",
        "— A community post reported by five different people is hidden from the feed until it has been reviewed. Reports from accounts created in the last few hours do reach the administrators, but they don't count towards those five — otherwise a post could be hidden with a batch of fresh accounts.",
        "— Reports are reviewed by the service's administrators. If you are being threatened or are in danger, contact the police first: we are not an emergency service.",
      ],
    },
    {
      title: "What happens when the rules are broken",
      body: [
        "Depending on how serious the violation is and whether it is repeated:",
        "— removal of a community post — by the community's editors or by the administrators;",
        "— for a group or community: a warning, exclusion from the main feed, deletion (a third active warning deletes the community);",
        "— for an account: a block, after which it can no longer be signed in to.",
        "For serious violations — threats to life, child sexual abuse, terrorism — the account is blocked without warning, and information may be passed on to law enforcement in accordance with the procedure established by law.",
        "An account created without an invitation works in a restricted mode for the first few hours after sign-up: this is protection against spam, not a punishment.",
      ],
    },
    {
      title: "If you disagree with a decision",
      body: [
        "Write to kisyandco@gmail.com: your username, what happened and why you believe the decision was wrong. We will review it and reply, usually within 30 days.",
      ],
    },
    {
      title: "Changes to the guidelines",
      body: [
        "These guidelines may change. When there are significant changes, the app will ask you to accept the new version the next time you sign in — you cannot use the service without doing so. The date of the last change is given at the top of the page.",
      ],
    },
  ],
};
