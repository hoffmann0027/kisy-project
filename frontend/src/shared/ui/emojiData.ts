// A curated emoji set for the picker (stage F). Kept compact and searchable
// by keyword rather than shipping the full Unicode set — covers the common
// chat needs across smileys, gestures, hearts, objects and symbols.
export interface EmojiCategory {
  id: string;
  label: string;
  emojis: { char: string; keywords: string }[];
}

export const EMOJI_CATEGORIES: EmojiCategory[] = [
  {
    id: "smileys",
    label: "Смайлы",
    emojis: [
      { char: "😀", keywords: "улыбка smile grin happy" },
      { char: "😁", keywords: "улыбка grin" },
      { char: "😂", keywords: "смех laugh joy слёзы" },
      { char: "🤣", keywords: "смех rofl хохот" },
      { char: "😊", keywords: "улыбка blush рад" },
      { char: "😍", keywords: "любовь love глаза сердце" },
      { char: "😘", keywords: "поцелуй kiss" },
      { char: "😎", keywords: "круто cool очки" },
      { char: "🤔", keywords: "думаю think размышление" },
      { char: "😐", keywords: "нейтрально neutral" },
      { char: "🙄", keywords: "закатить глаза eyeroll" },
      { char: "😴", keywords: "сон sleep спать" },
      { char: "😭", keywords: "плач cry рыдать" },
      { char: "😡", keywords: "злость angry гнев" },
      { char: "🥳", keywords: "праздник party ура" },
      { char: "🤯", keywords: "взрыв мозга mind blown" },
      { char: "😱", keywords: "шок scream страх" },
      { char: "🤗", keywords: "объятия hug" },
      { char: "😉", keywords: "подмигивание wink" },
      { char: "🙂", keywords: "улыбка slight smile" },
    ],
  },
  {
    id: "gestures",
    label: "Жесты",
    emojis: [
      { char: "👍", keywords: "лайк like thumbs up хорошо" },
      { char: "👎", keywords: "дизлайк dislike плохо" },
      { char: "👏", keywords: "аплодисменты clap браво" },
      { char: "🙏", keywords: "спасибо please молитва" },
      { char: "🤝", keywords: "рукопожатие handshake сделка" },
      { char: "✌️", keywords: "мир peace victory" },
      { char: "🤞", keywords: "удача fingers crossed" },
      { char: "👌", keywords: "ок okay отлично" },
      { char: "💪", keywords: "сила strong мышцы" },
      { char: "👋", keywords: "привет wave hello пока" },
      { char: "🤟", keywords: "рок love you" },
      { char: "☝️", keywords: "внимание point up" },
    ],
  },
  {
    id: "hearts",
    label: "Сердца",
    emojis: [
      { char: "❤️", keywords: "сердце love red heart" },
      { char: "🧡", keywords: "сердце orange" },
      { char: "💛", keywords: "сердце yellow" },
      { char: "💚", keywords: "сердце green" },
      { char: "💙", keywords: "сердце blue" },
      { char: "💜", keywords: "сердце purple" },
      { char: "🖤", keywords: "сердце black" },
      { char: "🤍", keywords: "сердце white" },
      { char: "💔", keywords: "разбитое broken heart" },
      { char: "💕", keywords: "два сердца two hearts" },
      { char: "🔥", keywords: "огонь fire круто" },
      { char: "✨", keywords: "искры sparkle" },
    ],
  },
  {
    id: "objects",
    label: "Объекты",
    emojis: [
      { char: "🎉", keywords: "праздник party хлопушка" },
      { char: "🎂", keywords: "торт cake день рождения" },
      { char: "🎁", keywords: "подарок gift" },
      { char: "💯", keywords: "сто 100 идеально" },
      { char: "✅", keywords: "галочка check готово" },
      { char: "❌", keywords: "крест cross нет" },
      { char: "⚠️", keywords: "внимание warning" },
      { char: "💡", keywords: "идея idea лампочка" },
      { char: "📌", keywords: "закрепить pin" },
      { char: "⭐", keywords: "звезда star" },
      { char: "☕", keywords: "кофе coffee" },
      { char: "🚀", keywords: "ракета rocket запуск" },
      { char: "💻", keywords: "компьютер laptop работа" },
      { char: "📎", keywords: "скрепка clip вложение" },
      { char: "⏰", keywords: "время time будильник" },
      { char: "💰", keywords: "деньги money" },
    ],
  },
];

export const ALL_EMOJIS = EMOJI_CATEGORIES.flatMap((c) => c.emojis);

export function searchEmojis(query: string): { char: string; keywords: string }[] {
  const q = query.trim().toLowerCase();
  if (!q) return [];
  return ALL_EMOJIS.filter((e) => e.keywords.toLowerCase().includes(q) || e.char === q);
}
