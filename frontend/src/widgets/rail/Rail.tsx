import { useLocation, useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { AlertBadge, Avatar, Badge, Logo } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useAuthStore } from "@shared/store/auth";
import { useCapabilities } from "@shared/lib/useCapabilities";
import { communitiesDestination, ratingOrFeed } from "@shared/lib/nav";
import { useNotifications } from "@entities/notification/queries";
import { useChats } from "@entities/chat/queries";
import { t } from "@shared/i18n";

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
  const caps = useCapabilities();

  if (!user) return null;

  // Rating for an account that has a board, the feed for one that has not —
  // decided in shared/lib/nav.ts so the rail and the tab bar cannot disagree
  // (audit D-11). The rail used to offer level 10 a Рейтинг button whose only
  // answer was "you are not in a clan".
  const third = ratingOrFeed(caps);
  // Groups now live under the "Сообщества" (communities) section; a group route
  // therefore highlights Сообщества, not Чаты.
  const onCommunities = communitiesDestination.match(pathname);
  const onChats = !third.match(pathname) && !onCommunities && !pathname.startsWith("/hub");

  return (
    <nav className="rail">
      <div className="rail__logo">
        <Logo size={34} />
      </div>
      <div className="rail__nav">
        <button
          className={cn("rail__item", third.match(pathname) && "rail__item--active")}
          title={third.label}
          onClick={() => navigate(third.to)}
        >
          <third.icon />
        </button>
        <button className={cn("rail__item", onChats && "rail__item--active")} title={t("hub.rail.chats")} onClick={() => navigate("/")}>
          <Icon.Chat />
          {chatUnread > 0 && (
            <span className="rail__item-badge">
              <Badge>{chatUnread > 9 ? "9+" : chatUnread}</Badge>
            </span>
          )}
        </button>
        <button
          className={cn("rail__item", onCommunities && "rail__item--active")}
          title={t("hub.rail.communities")}
          onClick={() => navigate("/communities")}
        >
          <Icon.Community />
        </button>
        <button
          className={cn("rail__item", pathname.startsWith("/hub") && "rail__item--active")}
          title={t("hub.rail.hub")}
          onClick={() => navigate("/hub")}
        >
          <Icon.Grid />
          <AlertBadge count={unread} label={t("hub.page.unread", { count: unread })} />
        </button>
      </div>
      <button className="rail__item" title={t("hub.rail.profile")} onClick={onProfile}>
        <Avatar name={user.displayName} url={user.avatarUrl} size={38} />
      </button>
      <button className="rail__item" title={t("hub.rail.logout")} onClick={() => void logout()}>
        <Icon.Logout />
      </button>
    </nav>
  );
}
