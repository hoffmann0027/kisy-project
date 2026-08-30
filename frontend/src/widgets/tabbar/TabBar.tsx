import { useLocation, useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { Icon } from "@shared/ui/icons";
import "./tabbar.css";

// Bottom navigation for the phone layout (design_handoff_kisy_mobile §6): a
// floating capsule with four tabs around a raised "orb" that opens the Hub.
// It replaces the 58px rail on narrow screens; the desktop rail is untouched.
//
// The handoff pictures Communities/Rating/Profile as placeholders because they
// were not redesigned yet — but those screens exist and work, so the tabs lead
// to the real ones.

interface Slot {
  key: string;
  label: string;
  icon: (p: { size?: number }) => React.ReactElement;
  to?: string;
  onSelect?: () => void;
  /** Paths that should light this slot up. */
  match: (path: string) => boolean;
}

interface Props {
  /** Opens the profile modal — profile is a dialog, not a route. */
  onProfile: () => void;
}

export function TabBar({ onProfile }: Props) {
  const navigate = useNavigate();
  const { pathname } = useLocation();

  const slots: Slot[] = [
    {
      key: "messages",
      label: "Сообщения",
      icon: Icon.Chat,
      to: "/",
      match: (p) => p === "/" || p.startsWith("/chat/"),
    },
    {
      key: "communities",
      label: "Сообщества",
      icon: Icon.Community,
      to: "/communities",
      match: (p) => p.startsWith("/communities") || p.startsWith("/group/"),
    },
    {
      key: "rating",
      label: "Рейтинг",
      icon: Icon.Trophy,
      to: "/rating",
      match: (p) => p.startsWith("/rating"),
    },
    {
      key: "profile",
      label: "Профиль",
      icon: Icon.User,
      onSelect: onProfile,
      match: () => false, // a modal, never an active route
    },
  ];

  // The chat screen takes the full height — the bar would only cover it.
  // Both a private chat and a group open a conversation on mobile.
  if (pathname.startsWith("/chat/") || pathname.startsWith("/group/")) return null;

  const hubActive = pathname.startsWith("/hub");
  const go = (slot: Slot) => (slot.to ? navigate(slot.to) : slot.onSelect?.());

  // The orb sits between the second and third tab.
  const left = slots.slice(0, 2);
  const right = slots.slice(2);

  const renderSlot = (slot: Slot) => {
    const active = !hubActive && slot.match(pathname);
    return (
      <button
        key={slot.key}
        type="button"
        className={cn("tabbar__slot", active && "tabbar__slot--active")}
        aria-label={slot.label}
        aria-current={active ? "page" : undefined}
        onClick={() => go(slot)}
      >
        <span className="tabbar__glow" aria-hidden="true" />
        <span className="tabbar__icon">
          <slot.icon size={24} />
        </span>
      </button>
    );
  };

  return (
    <nav className="tabbar" aria-label="Основная навигация">
      {left.map(renderSlot)}

      <button
        type="button"
        className={cn("tabbar__orb", hubActive && "tabbar__orb--active")}
        aria-label="Хаб"
        aria-current={hubActive ? "page" : undefined}
        onClick={() => navigate("/hub")}
      >
        <span className="tabbar__orb-face">
          <Icon.Grid size={24} />
        </span>
      </button>

      {right.map(renderSlot)}
    </nav>
  );
}
