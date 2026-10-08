// Colours the dashboard's charts are drawn in. Recharts writes colours into
// SVG attributes, where a CSS variable is not resolved, so the theme's accent
// and inks are read once from the stylesheet (useThemeInk) and the fixed
// series colours are chosen to read on the light themes and the dark ones.
import { useEffect, useState } from "react";
import type { ProjectState } from "@entities/rating/model";

export const MONEY = { income: "#22c55e", expense: "#f43f5e" } as const;

export const STATE_COLOR: Record<ProjectState, string> = {
  working: "#22c55e",
  open: "#3b82f6",
  idle: "#f59e0b",
  done: "#a78bfa",
};

/** One colour per project in the profit-share donut, in a fixed order. */
export const SHARE = ["#7c5cff", "#22c55e", "#3b82f6", "#f59e0b", "#f43f5e", "#14b8a6", "#ec4899", "#a3e635"];

export interface ThemeInk {
  accent: string;
  grid: string;
  axis: string;
}

const FALLBACK: ThemeInk = { accent: "#7c5cff", grid: "rgba(128,128,128,0.2)", axis: "#8a8a99" };

function readInk(): ThemeInk {
  if (typeof document === "undefined") return FALLBACK;
  const css = getComputedStyle(document.documentElement);
  const v = (name: string, fallback: string) => css.getPropertyValue(name).trim() || fallback;
  return { accent: v("--acc", FALLBACK.accent), grid: v("--divider", FALLBACK.grid), axis: v("--dim2", FALLBACK.axis) };
}

/** The current theme's accent and muted inks; follows a theme switch. */
export function useThemeInk(): ThemeInk {
  const [ink, setInk] = useState<ThemeInk>(readInk);
  useEffect(() => {
    if (typeof MutationObserver === "undefined") return;
    const mo = new MutationObserver(() => setInk(readInk()));
    mo.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme", "class"] });
    return () => mo.disconnect();
  }, []);
  return ink;
}
