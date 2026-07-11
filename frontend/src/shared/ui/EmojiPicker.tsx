// Emoji picker (stage F): categories, keyword search and a "recent" row
// persisted in localStorage. Used from the composer (insert into text) and
// from a message's reaction menu (react with any emoji, not just the 5
// quick ones). Closes on outside click / Esc.
import { useEffect, useMemo, useRef, useState } from "react";
import { EMOJI_CATEGORIES, searchEmojis } from "./emojiData";

const RECENT_KEY = "kisy-emoji-recent";
const RECENT_MAX = 24;

function loadRecent(): string[] {
  try {
    const raw = JSON.parse(localStorage.getItem(RECENT_KEY) ?? "[]");
    return Array.isArray(raw) ? raw.filter((x) => typeof x === "string").slice(0, RECENT_MAX) : [];
  } catch {
    return [];
  }
}

export function pushRecentEmoji(char: string) {
  const next = [char, ...loadRecent().filter((c) => c !== char)].slice(0, RECENT_MAX);
  localStorage.setItem(RECENT_KEY, JSON.stringify(next));
}

interface Props {
  onPick: (char: string) => void;
  onClose: () => void;
  /** Anchor className to skip in the outside-click check (the toggle button). */
  ignoreSelector?: string;
}

export function EmojiPicker({ onPick, onClose, ignoreSelector }: Props) {
  const [query, setQuery] = useState("");
  const [recent] = useState(loadRecent);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      const t = e.target as HTMLElement;
      if (rootRef.current?.contains(t)) return;
      if (ignoreSelector && t.closest(ignoreSelector)) return;
      onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [onClose, ignoreSelector]);

  const results = useMemo(() => searchEmojis(query), [query]);

  const pick = (char: string) => {
    pushRecentEmoji(char);
    onPick(char);
  };

  return (
    <div className="emojipick" ref={rootRef} role="dialog" aria-label="Выбор эмодзи">
      <input
        className="emojipick__search ui-input"
        placeholder="Поиск эмодзи"
        autoFocus
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div className="emojipick__scroll">
        {query ? (
          <section className="emojipick__section">
            <div className="emojipick__grid">
              {results.map((e) => (
                <button key={e.char} className="emojipick__emoji" onClick={() => pick(e.char)}>
                  {e.char}
                </button>
              ))}
              {results.length === 0 && <div className="emojipick__empty">Ничего не найдено</div>}
            </div>
          </section>
        ) : (
          <>
            {recent.length > 0 && (
              <section className="emojipick__section">
                <div className="emojipick__label">Недавние</div>
                <div className="emojipick__grid">
                  {recent.map((c) => (
                    <button key={`r-${c}`} className="emojipick__emoji" onClick={() => pick(c)}>
                      {c}
                    </button>
                  ))}
                </div>
              </section>
            )}
            {EMOJI_CATEGORIES.map((cat) => (
              <section key={cat.id} className="emojipick__section">
                <div className="emojipick__label">{cat.label}</div>
                <div className="emojipick__grid">
                  {cat.emojis.map((e) => (
                    <button key={e.char} className="emojipick__emoji" onClick={() => pick(e.char)}>
                      {e.char}
                    </button>
                  ))}
                </div>
              </section>
            ))}
          </>
        )}
      </div>
    </div>
  );
}
