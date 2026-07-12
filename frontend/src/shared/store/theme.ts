import { create } from "zustand";
import { persist } from "zustand/middleware";

// Two visual themes from the design handoff: `glass` (light, frosted) and
// `luce` (dark, cold aluminium/graphite + red accent). The choice is
// persisted so it survives reloads (key `kisy-theme`); the active theme is
// reflected as `data-theme` on <html>, which selects the token set in
// theme.css. Default is `glass`.
export type Theme = "glass" | "luce";

interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: "glass",
      setTheme: (theme) => set({ theme }),
      toggle: () => set((s) => ({ theme: s.theme === "glass" ? "luce" : "glass" })),
    }),
    { name: "kisy-theme" },
  ),
);
