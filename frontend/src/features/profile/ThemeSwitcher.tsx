// Shared by the profile dialog and the phone drawer. It lives on its own
// because those are two entry points to the same control and the knob must
// not be copied into each of them.
import { useThemeStore, type Theme } from "@shared/store/theme";

// Theme switcher (design handoff): a physical rotary knob ("manettino")
// centered between two columns of labels. The indicator rests at 3 o'clock and
// the knob rotates to point at the active label — the left column (Орбита /
// Стекло / Luce / Аврора) is pointed at by rotating left, the right column
// (Cyber / Windows 95 / Matrix) by rotating right. Clicking the knob cycles
// through all seven; clicking a label selects it directly. The angles follow
// the label rows, so the left column's four seats sit closer together than the
// right column's three.
type ThemeOption = { id: Theme; label: string; angle: number; col: "left" | "right" };
const THEME_OPTIONS: ThemeOption[] = [
  { id: "orbit", label: "Орбита", angle: -150, col: "left" },
  { id: "glass", label: "Стекло", angle: -170, col: "left" },
  { id: "luce", label: "Luce", angle: 170, col: "left" },
  { id: "aurora", label: "Аврора", angle: 150, col: "left" },
  { id: "cyber", label: "Cyber", angle: -30, col: "right" },
  { id: "xp", label: "Windows 95", angle: 0, col: "right" },
  { id: "matrix", label: "Matrix", angle: 30, col: "right" },
];

function GearIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" />
    </svg>
  );
}

export function ThemeSwitcher() {
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const cycleTheme = useThemeStore((s) => s.cycleTheme);
  const angle = THEME_OPTIONS.find((o) => o.id === theme)?.angle ?? 0;

  const labelBtn = (o: ThemeOption) => (
    <button
      key={o.id}
      type="button"
      className={`theme-knob__label${theme === o.id ? " theme-knob__label--active" : ""}`}
      onClick={() => setTheme(o.id)}
    >
      {o.label}
    </button>
  );

  return (
    <div className="profile-section">
      <div className="profile-section__label">
        <GearIcon />
        Оформление
      </div>
      <div className="theme-knob">
        <div className="theme-knob__labels theme-knob__labels--left">
          {THEME_OPTIONS.filter((o) => o.col === "left").map(labelBtn)}
        </div>
        <button type="button" className="theme-knob__dial" onClick={cycleTheme} aria-label="Переключить тему">
          <span className="theme-knob__face" style={{ transform: `rotate(${angle}deg)` }}>
            <span className="theme-knob__dimple" />
            <span className="theme-knob__pointer" />
          </span>
        </button>
        <div className="theme-knob__labels theme-knob__labels--right">
          {THEME_OPTIONS.filter((o) => o.col === "right").map(labelBtn)}
        </div>
      </div>
    </div>
  );
}