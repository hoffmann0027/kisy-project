import { lazy, Suspense, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import "./messenger.css";
import { Rail } from "@widgets/rail/Rail";
import { ChatList } from "@widgets/chat-list/ChatList";
import { Conversation } from "@widgets/conversation/Conversation";
import { GroupView } from "@widgets/conversation/GroupView";
import { AppDrawer } from "@widgets/drawer/AppDrawer";
import { Icon } from "@shared/ui/icons";
import { formatRelative } from "@shared/lib/format";
import type { Chat, Group } from "@shared/api/types";
import { useChats } from "@entities/chat/queries";
import { useGroups } from "@entities/group/queries";
import { usePresenceStore } from "@shared/store/presence";

// Dialogs are loaded the first time they are opened, rather than sitting in the
// entry chunk to be parsed on every cold start. Six more used to be declared
// here and never opened — their working copies live in the Hub (audit C-08).
const NewChatModal = lazy(() => import("@features/new-chat/NewChatModal").then((m) => ({ default: m.NewChatModal })));
const NewGroupModal = lazy(() => import("@features/new-chat/NewGroupModal").then((m) => ({ default: m.NewGroupModal })));
const ProfileModal = lazy(() => import("@features/profile/ProfileModal").then((m) => ({ default: m.ProfileModal })));

export function MessengerPage() {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { chatId, groupId } = useParams();
  // Groups moved under "Сообщества": the middle column shows groups on the
  // communities route (and whenever a group is open), private chats otherwise.
  const view: "chats" | "communities" =
    pathname.startsWith("/communities") || pathname.startsWith("/group/") ? "communities" : "chats";
  const { data: chats } = useChats();
  const { data: groups } = useGroups();
  const online = usePresenceStore((s) => s.online);

  const [newChat, setNewChat] = useState(false);
  const [newGroup, setNewGroup] = useState(false);
  const [profile, setProfile] = useState(false);
  // Phone layout: the side drawer opened from the header avatar.
  const [drawer, setDrawer] = useState(false);

  const activeChat: Chat | undefined = chats?.find((c) => c.id === chatId);
  const activeGroup: Group | undefined = groups?.find((g) => g.id === groupId);
  const activeId = chatId ?? groupId ?? null;

  const selectChat = (chat: Chat) => navigate(`/chat/${chat.id}`);
  const selectGroup = (group: Group) => navigate(`/group/${group.id}`);

  const other = activeChat?.otherUser;
  const chatOnline = other ? online.has(other.id) || other.status === "online" : false;
  const offlineLabel = other?.lastSeen ? `был(а) в сети ${formatRelative(other.lastSeen)}` : "не в сети";

  return (
    <div className={activeId ? "msgr msgr--chat-open" : "msgr"}>
      <Rail onProfile={() => setProfile(true)} />

      <ChatList
        view={view}
        onOpenDrawer={() => setDrawer(true)}
        activeId={activeId}
        onSelect={selectChat}
        onSelectGroup={selectGroup}
        onNewChat={() => setNewChat(true)}
        onNewGroup={() => setNewGroup(true)}
      />

      {activeGroup ? (
        <GroupView key={activeGroup.id} group={activeGroup} />
      ) : activeChat && other ? (
        <Conversation
          key={activeChat.id}
          target={{
            chatType: "private",
            chatId: activeChat.id,
            title: other.displayName,
            verified: !!other.verifiedAt,
            avatarName: other.displayName,
            avatarUrl: other.avatarUrl,
            online: chatOnline,
            offlineLabel,
            otherLastReadAt: activeChat.otherLastReadAt,
            peerUserId: other.id,
          }}
        />
      ) : (
        <div className="ui-empty">
          <div style={{ opacity: 0.4 }}>
            <Icon.Chat size={64} />
          </div>
          <div style={{ fontSize: 17, fontWeight: 600, color: "var(--color-text-primary)" }}>Выберите чат</div>
          <div>Откройте диалог, группу или начните новый</div>
        </div>
      )}

      <Suspense fallback={null}>
      {newChat && <NewChatModal open onClose={() => setNewChat(false)} onOpened={selectChat} />}
      {newGroup && <NewGroupModal open onClose={() => setNewGroup(false)} onCreated={selectGroup} />}
      {profile && <ProfileModal open onClose={() => setProfile(false)} />}
      </Suspense>

      <AppDrawer open={drawer} onClose={() => setDrawer(false)} />
    </div>
  );
}
