import { create } from "zustand";
import { persist } from "zustand/middleware";

// Four visual themes from the design handoff:
//   `glass`  — light, frosted, violet accent
//   `luce`   — dark, cold aluminium/graphite + red accent
//   `aurora` — light, vivid pink→violet→indigo gradients, magenta accent
//   `cyber`  — dark neon/cyberpunk, cyan→magenta→violet glow, cyan accent
// The choice is persisted so it survives reloads (key `kisy-theme`); the
// active theme is reflected as `data-theme` on <html>, which selects the token
// set in theme.css. Default is `glass`; an unknown persisted value falls back
// to `glass`.
export type Theme = "glass" | "luce" | "aurora" | "cyber";

export const THEME_ORDER: Theme[] = ["glass", "luce", "aurora", "cyber"];

const isTheme = (v: unknown): v is Theme => THEME_ORDER.includes(v as Theme);

interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  // Advance through the themes in a cycle: glass → luce → aurora → cyber → glass.
  cycleTheme: () => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: "glass",
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
        return { ...current, ...p, theme: isTheme(p?.theme) ? p!.theme : "glass" };
      },
    },
  ),
);
