import { create } from "zustand";
import { persist } from "zustand/middleware";

// Seven visual themes:
//   `orbit`  — dark indigo/violet, the phone redesign's palette; the default
//   `glass`  — light, frosted, violet accent
//   `luce`   — dark, cold aluminium/graphite + red accent
//   `aurora` — light, vivid pink→violet→indigo gradients, magenta accent
//   `cyber`  — dark neon/cyberpunk, cyan→magenta→violet glow, cyan accent
//   `xp`     — retro Windows 95 (teal desktop, grey bevels); label "Windows 95"
//   `matrix` — phosphor terminal, acid-green on black, monospace
// The choice is persisted so it survives reloads (key `kisy-theme`); the
// active theme is reflected as `data-theme` on <html>, which selects the token
// set in theme.css. Default is `orbit`; an unknown persisted value falls back
// to `orbit`. (`xp` key kept for persist compatibility; UI label is "Windows 95".)
export type Theme = "orbit" | "glass" | "luce" | "aurora" | "cyber" | "xp" | "matrix";

export const THEME_ORDER: Theme[] = ["orbit", "glass", "luce", "aurora", "cyber", "xp", "matrix"];

const isTheme = (v: unknown): v is Theme => THEME_ORDER.includes(v as Theme);

interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  // Advance through the themes in a cycle: orbit → glass → luce → aurora → cyber → xp → matrix → orbit.
  cycleTheme: () => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: "orbit",
      setTheme: (theme) => set({ theme }),
      cycleTheme: () =>
        set((s) => {
          const i = THEME_ORDER.indexOf(s.theme);
          return { theme: THEME_ORDER[(i + 1) % THEME_ORDER.length] };
        }),
    }),
    {
      name: "kisy-theme",
      // Guard against a stale/unknown persisted theme (e.g. the removed
      // "carbon") — fall back to the default rather than a broken UI.
      merge: (persisted, current) => {
        const p = persisted as Partial<ThemeState> | undefined;
        return { ...current, ...p, theme: isTheme(p?.theme) ? p!.theme : "orbit" };
      },
    },
  ),
);
