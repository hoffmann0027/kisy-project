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
  onNotifications: () => void;
  onFeedback: () => void;
  onNotes: () => void;
  onConditions: () => void;
  onVoting: () => void;
  onCalls: () => void;
}

export function Rail({ onProfile, onNotifications, onFeedback, onNotes, onConditions, onVoting, onCalls }: Props) {
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
  const canRating = user.roleLevel <= RATING_MAX_LEVEL;
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
        <button className={cn("rail__item", !onRating && "rail__item--active")} title="Чаты" onClick={() => navigate("/")}>
          <Icon.Chat />
          {chatUnread > 0 && (
            <span className="rail__item-badge">
              <Badge>{chatUnread > 9 ? "9+" : chatUnread}</Badge>
            </span>
          )}
        </button>
        <button className="rail__item" title="Уведомления" onClick={onNotifications}>
          <Icon.Bell />
          {unread > 0 && (
            <span className="rail__item-badge">
              <Badge>{unread > 9 ? "9+" : unread}</Badge>
            </span>
          )}
        </button>
        {user.roleLevel === 1 && (
          <button className="rail__item" title="Администрирование" onClick={() => navigate("/admin")}>
            <Icon.Shield />
          </button>
        )}
      </div>
      <button className="rail__item" title="Голосование" onClick={onVoting}>
        <Icon.Vote />
      </button>
      <button className="rail__item" title="Условия повышения уровня" onClick={onConditions}>
        <Icon.Levels />
      </button>
      <button className="rail__item" title="История звонков" onClick={onCalls}>
        <Icon.Phone />
      </button>
      <button className="rail__item" title="Заметки" onClick={onNotes}>
        <Icon.Note />
      </button>
      <button className="rail__item" title="Отзывы и предложения" onClick={onFeedback}>
        <Icon.Feedback />
      </button>
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
