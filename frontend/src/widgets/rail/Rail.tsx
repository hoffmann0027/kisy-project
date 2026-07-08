import { useLocation, useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { Avatar, Badge, Logo } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useAuthStore } from "@shared/store/auth";
import { useNotifications } from "@entities/notification/queries";
import { useChats } from "@entities/chat/queries";

interface Props {
  onProfile: () => void;
  onNotifications: () => void;
  onFeedback: () => void;
  onNotes: () => void;
  onConditions: () => void;
  onVoting: () => void;
}

export function Rail({ onProfile, onNotifications, onFeedback, onNotes, onConditions, onVoting }: Props) {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { data: notif } = useNotifications();
  const { data: chats } = useChats();
  const unread = notif?.unreadCount ?? 0;
  const chatUnread = chats?.reduce((sum, c) => sum + c.unreadCount, 0) ?? 0;

  if (!user) return null;

  const onRating = pathname.startsWith("/rating");

  return (
    <nav className="rail">
      <div className="rail__logo">
        <Logo size={34} />
      </div>
      <div className="rail__nav">
        <button className={cn("rail__item", onRating && "rail__item--active")} title="Рейтинг" onClick={() => navigate("/rating")}>
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
    </nav>
  );
}
