import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { createPortal } from "react-dom";
import { cn } from "@shared/lib/cn";
import { Avatar } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useAuthStore } from "@shared/store/auth";
import { roleLabel } from "@shared/api/types";
import "./drawer.css";

// Side drawer for the phone layout (design_handoff_kisy_mobile §7), opened by
// the avatar in the Messages header. It carries the sections that do not fit
// the four tabs; the Hub covers the rest.

interface Props {
  open: boolean;
  onClose: () => void;
  /** Sections that live in modals rather than routes. */
  onOpen: (what: "notifications" | "notes" | "feedback" | "profile") => void;
  unread?: number;
}

export function AppDrawer({ open, onClose, onOpen, unread = 0 }: Props) {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open || !user) return null;

  const go = (to: string) => {
    navigate(to);
    onClose();
  };
  const open_ = (what: Parameters<Props["onOpen"]>[0]) => {
    onOpen(what);
    onClose();
  };

  const items = [
    { key: "messages", label: "Сообщения", icon: Icon.Chat, active: pathname === "/" || pathname.startsWith("/chat/"), run: () => go("/") },
    { key: "communities", label: "Сообщества", icon: Icon.Community, active: pathname.startsWith("/communities") || pathname.startsWith("/group/"), run: () => go("/communities") },
    { key: "hub", label: "Хаб", icon: Icon.Grid, active: pathname.startsWith("/hub"), run: () => go("/hub") },
    { key: "rating", label: "Рейтинг проектов", icon: Icon.Trophy, active: pathname.startsWith("/rating"), run: () => go("/rating") },
    { key: "notifications", label: "Уведомления", icon: Icon.Bell, badge: unread, run: () => open_("notifications") },
    { key: "notes", label: "Заметки", icon: Icon.Note, run: () => open_("notes") },
    { key: "feedback", label: "Отзывы и предложения", icon: Icon.Feedback, run: () => open_("feedback") },
    { key: "profile", label: "Настройки профиля", icon: Icon.Settings, run: () => open_("profile") },
  ];

  // Portal to <body>: the drawer must cover the whole viewport, not be clipped
  // by a panel with overflow:hidden.
  return createPortal(
    <div className="drawer" role="dialog" aria-modal="true" aria-label="Меню">
      <div className="drawer__scrim" onClick={onClose} />

      <aside className="drawer__panel">
        <header className="drawer__head">
          <Avatar name={user.displayName} url={user.avatarUrl} size={46} />
          <div className="drawer__who">
            <div className="drawer__name">{user.displayName}</div>
            <div className="drawer__role">{roleLabel(user.roleLevel)}</div>
          </div>
          <button type="button" className="drawer__close" aria-label="Закрыть" onClick={onClose}>
            <Icon.X size={20} />
          </button>
        </header>

        <nav className="drawer__list">
          {items.map((it) => (
            <button
              key={it.key}
              type="button"
              className={cn("drawer__item", it.active && "drawer__item--active")}
              onClick={it.run}
            >
              <span className="drawer__item-icon">
                <it.icon size={20} />
              </span>
              <span className="drawer__item-label">{it.label}</span>
              {it.badge ? (
                <span className="drawer__item-badge">{it.badge > 99 ? "99+" : it.badge}</span>
              ) : (
                <span className="drawer__item-chevron">
                  <Icon.Chevron size={18} />
                </span>
              )}
            </button>
          ))}
        </nav>

        <footer className="drawer__foot">
          <button type="button" className="drawer__logout" onClick={() => void logout()}>
            <Icon.Logout size={20} />
            <span>Выйти</span>
          </button>
        </footer>
      </aside>
    </div>,
    document.body,
  );
}
