import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { Avatar, Badge, Button, Logo, Modal } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useAuthStore } from "@shared/store/auth";
import { useNotifications } from "@entities/notification/queries";
import { useChats } from "@entities/chat/queries";

// Rating (the clan board) is open to clearance levels 1–9; the weakest
// level (10) is "not in a clan" and gets a small popup instead of access.
const RATING_MAX_LEVEL = 9;

interface Props {
  onProfile: () => void;
}

// Desktop counterpart of the phone tab bar. Same rule as the drawer: nothing
// here may duplicate the hub. Notifications, votes, notes, feedback, level
// conditions, call history and admin all moved there — the rail keeps the
// first-level destinations plus the profile and sign-out.
export function Rail({ onProfile }: Props) {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { data: notif } = useNotifications();
  const { data: chats } = useChats();
  const unread = notif?.unreadCount ?? 0;
  const chatUnread = chats?.reduce((sum, c) => sum + c.unreadCount, 0) ?? 0;
  const [noClan, setNoClan] = useState(false);

  if (!user) return null;

  const onRating = pathname.startsWith("/rating");
  // Groups now live under the "Сообщества" (communities) section; a group route
  // therefore highlights Сообщества, not Чаты.
  const onCommunities = pathname.startsWith("/communities") || pathname.startsWith("/group/");
  const onChats = !onRating && !onCommunities;
  // An account with no level is not at the bottom of the hierarchy, it is
  // outside it, so no threshold admits it.
  const canRating = user.roleLevel !== null && user.roleLevel <= RATING_MAX_LEVEL;
  const openRating = () => (canRating ? navigate("/rating") : setNoClan(true));

  return (
    <nav className="rail">
      <div className="rail__logo">
        <Logo size={34} />
      </div>
      <div className="rail__nav">
        <button className={cn("rail__item", onRating && "rail__item--active")} title="Рейтинг" onClick={openRating}>
          <Icon.Trophy />
        </button>
        <button className={cn("rail__item", onChats && "rail__item--active")} title="Чаты" onClick={() => navigate("/")}>
          <Icon.Chat />
          {chatUnread > 0 && (
            <span className="rail__item-badge">
              <Badge>{chatUnread > 9 ? "9+" : chatUnread}</Badge>
            </span>
          )}
        </button>
        <button
          className={cn("rail__item", onCommunities && "rail__item--active")}
          title="Сообщества"
          onClick={() => navigate("/communities")}
        >
          <Icon.Community />
        </button>
        <button
          className={cn("rail__item", pathname.startsWith("/hub") && "rail__item--active")}
          title="Хаб"
          onClick={() => navigate("/hub")}
        >
          <Icon.Grid />
          {unread > 0 && (
            <span className="rail__item-badge">
              <Badge>{unread > 9 ? "9+" : unread}</Badge>
            </span>
          )}
        </button>
      </div>
      <button className="rail__item" title="Профиль" onClick={onProfile}>
        <Avatar name={user.displayName} url={user.avatarUrl} size={38} />
      </button>
      <button className="rail__item" title="Выйти" onClick={() => void logout()}>
        <Icon.Logout />
      </button>

      <Modal open={noClan} title="Рейтинг" onClose={() => setNoClan(false)}>
        <p style={{ margin: 0, color: "var(--color-text-secondary)" }}>Вы не состоите в клане.</p>
        <Button variant="secondary" onClick={() => setNoClan(false)}>
          Понятно
        </Button>
      </Modal>
    </nav>
  );
}
