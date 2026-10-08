// Emoji picker (stage F): categories, keyword search and a "recent" row
// persisted in localStorage. Used from the composer (insert into text) and
// from a message's reaction menu (react with any emoji, not just the 5
// quick ones). Closes on outside click / Esc.
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useBackHandler } from "@shared/lib/backStack";
import { t } from "@shared/i18n";
import { EMOJI_CATEGORIES, searchEmojis } from "./emojiData";
import "./EmojiPicker.css";

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
  /**
   * Put the cursor in the search field on open. Right when typing is the
   * point (the composer); wrong for a reaction on a phone, where focusing
   * raises the keyboard over the very grid the reader came to tap.
   */
  autoFocusSearch?: boolean;
  /**
   * Float next to this element instead of sitting above the call site's
   * wrapper: rendered at the page root, opened on whichever side has room,
   * kept inside the window. A message near the top of a chat that cannot
   * scroll any higher otherwise gets a picker cut off by the chat's edge.
   */
  anchor?: HTMLElement | null;
}

const GAP = 8;
const MARGIN = 8;

/** Where a floating picker of size w×h goes next to `r` in a vw×vh window. */
export function floatingPosition(r: { top: number; bottom: number; left: number; width: number }, w: number, h: number, vw: number, vh: number): { top: number; left: number } {
  const roomAbove = r.top - GAP - MARGIN;
  const roomBelow = vh - r.bottom - GAP - MARGIN;
  // Above by default, like the rest of the app's popovers; below when only
  // there it fits; otherwise on the roomier side, clamped into the window.
  const goAbove = roomAbove >= h || (roomBelow < h && roomAbove >= roomBelow);
  const top = goAbove ? r.top - GAP - h : r.bottom + GAP;
  const left = r.left + r.width / 2 - w / 2;
  return {
    top: Math.max(MARGIN, Math.min(top, vh - h - MARGIN)),
    left: Math.max(MARGIN, Math.min(left, vw - w - MARGIN)),
  };
}

export function EmojiPicker({ onPick, onClose, ignoreSelector, autoFocusSearch = true, anchor }: Props) {
  // Mounted only while open, so it always claims the back gesture.
  useBackHandler(true, onClose);
  const [query, setQuery] = useState("");
  const [recent] = useState(loadRecent);
  const rootRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);

  // Not the autoFocus attribute: focusing scrolls every scrollable ancestor
  // to bring the field into view, and the whole chat slid sideways with it.
  useEffect(() => {
    if (autoFocusSearch) searchRef.current?.focus({ preventScroll: true });
  }, [autoFocusSearch]);

  // Measured before paint, so the picker never flashes in the wrong place.
  useLayoutEffect(() => {
    const el = rootRef.current;
    if (!anchor || !el) return;
    setPos(floatingPosition(anchor.getBoundingClientRect(), el.offsetWidth, el.offsetHeight, window.innerWidth, window.innerHeight));
  }, [anchor]);

  // A floating picker does not move with what it points at: scrolling the
  // chat or resizing the window closes it rather than leave it adrift.
  useEffect(() => {
    if (!anchor) return;
    const onScroll = (e: Event) => {
      if (e.target instanceof Node && rootRef.current?.contains(e.target)) return;
      onClose();
    };
    document.addEventListener("scroll", onScroll, true);
    window.addEventListener("resize", onClose);
    return () => {
      document.removeEventListener("scroll", onScroll, true);
      window.removeEventListener("resize", onClose);
    };
  }, [anchor, onClose]);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (rootRef.current?.contains(target)) return;
      if (ignoreSelector && target.closest(ignoreSelector)) return;
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

  const panel = (
    <div
      className={anchor ? "emojipick emojipick--floating" : "emojipick"}
      ref={rootRef}
      role="dialog"
      aria-label={t("common.emoji.picker")}
      style={anchor ? (pos ?? { visibility: "hidden" }) : undefined}
    >
      <input
        ref={searchRef}
        className="emojipick__search ui-input"
        placeholder={t("common.emoji.search")}
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
              {results.length === 0 && <div className="emojipick__empty">{t("common.emoji.noResults")}</div>}
            </div>
          </section>
        ) : (
          <>
            {recent.length > 0 && (
              <section className="emojipick__section">
                <div className="emojipick__label">{t("common.emoji.recent")}</div>
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
                <div className="emojipick__label">{t(cat.labelKey)}</div>
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
  return anchor ? createPortal(panel, document.body) : panel;
}
