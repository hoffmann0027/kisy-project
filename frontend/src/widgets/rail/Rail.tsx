import { useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { Avatar, Badge, Logo } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useAuthStore } from "@shared/store/auth";
import { useNotifications } from "@entities/notification/queries";

interface Props {
  onProfile: () => void;
  onNotifications: () => void;
}

export function Rail({ onProfile, onNotifications }: Props) {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const { data: notif } = useNotifications();
  const unread = notif?.unreadCount ?? 0;

  if (!user) return null;

  return (
    <nav className="rail">
      <div className="rail__logo">
        <Logo size={34} />
      </div>
      <div className="rail__nav">
        <button className={cn("rail__item", "rail__item--active")} title="Чаты" onClick={() => navigate("/")}>
          <Icon.Chat />
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
      <button className="rail__item" title="Профиль" onClick={onProfile}>
        <Avatar name={user.displayName} url={user.avatarUrl} size={38} />
      </button>
      <button className="rail__item" title="Выйти" onClick={() => void logout()}>
        <Icon.Logout />
      </button>
    </nav>
  );
}
