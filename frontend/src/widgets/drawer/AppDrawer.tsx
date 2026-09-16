import { useEffect } from "react";
import { createPortal } from "react-dom";
import { Avatar } from "@shared/ui";
import { ThemeSwitcher } from "@features/profile/ThemeSwitcher";
import { Icon } from "@shared/ui/icons";
import { useAuthStore } from "@shared/store/auth";
import { useBackHandler } from "@shared/lib/backStack";
import { roleLabel } from "@shared/api/types";
import "./drawer.css";
import { VerifiedName } from "@shared/ui/VerifiedBadge";

// Side drawer for the phone layout (design_handoff_kisy_mobile §7), opened by
// the avatar in the Messages header. It carries the sections that do not fit
// the four tabs; the Hub covers the rest.

interface Props {
  open: boolean;
  onClose: () => void;
}

export function AppDrawer({ open, onClose }: Props) {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  useBackHandler(open, onClose);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open || !user) return null;



  // Portal to <body>: the drawer must cover the whole viewport, not be clipped
  // by a panel with overflow:hidden.
  return createPortal(
    <div className="drawer" role="dialog" aria-modal="true" aria-label="Меню">
      <div className="drawer__scrim" onClick={onClose} />

      <aside className="drawer__panel">
        <header className="drawer__head">
          <Avatar name={user.displayName} url={user.avatarUrl} size={46} />
          <div className="drawer__who">
            <div className="drawer__name">
              <VerifiedName name={user.displayName} verified={!!user.verifiedAt} />
            </div>
            <div className="drawer__role">{roleLabel(user.roleLevel)}</div>
          </div>
          <button type="button" className="drawer__close" aria-label="Закрыть" onClick={onClose}>
            <Icon.X size={20} />
          </button>
        </header>

        {/* Every destination that used to live here is reachable from the tab
            bar or the hub; keeping copies meant three places to update and
            three places to disagree. What stays is what has no other home. */}
        <div className="drawer__theme">
          <ThemeSwitcher />
        </div>

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
