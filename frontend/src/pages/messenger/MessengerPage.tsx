import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import "./messenger.css";
import { Rail } from "@widgets/rail/Rail";
import { ChatList } from "@widgets/chat-list/ChatList";
import { Conversation } from "@widgets/conversation/Conversation";
import { GroupView } from "@widgets/conversation/GroupView";
import { NewChatModal } from "@features/new-chat/NewChatModal";
import { NewGroupModal } from "@features/new-chat/NewGroupModal";
import { ProfileModal } from "@features/profile/ProfileModal";
import { NotificationsModal } from "@features/notifications/NotificationsModal";
import { FeedbackModal } from "@features/feedback/FeedbackModal";
import { Icon } from "@shared/ui/icons";
import { formatRelative } from "@shared/lib/format";
import type { Chat, Group } from "@shared/api/types";
import { useChats } from "@entities/chat/queries";
import { useGroups } from "@entities/group/queries";
import { usePresenceStore } from "@shared/store/presence";

export function MessengerPage() {
  const navigate = useNavigate();
  const { chatId, groupId } = useParams();
  const { data: chats } = useChats();
  const { data: groups } = useGroups();
  const online = usePresenceStore((s) => s.online);

  const [newChat, setNewChat] = useState(false);
  const [newGroup, setNewGroup] = useState(false);
  const [profile, setProfile] = useState(false);
  const [notifications, setNotifications] = useState(false);
  const [feedback, setFeedback] = useState(false);

  const activeChat: Chat | undefined = chats?.find((c) => c.id === chatId);
  const activeGroup: Group | undefined = groups?.find((g) => g.id === groupId);
  const activeId = chatId ?? groupId ?? null;

  const selectChat = (chat: Chat) => navigate(`/chat/${chat.id}`);
  const selectGroup = (group: Group) => navigate(`/group/${group.id}`);

  const other = activeChat?.otherUser;
  const chatOnline = other ? online.has(other.id) || other.status === "online" : false;
  const offlineLabel = other?.lastSeen ? `был(а) в сети ${formatRelative(other.lastSeen)}` : "не в сети";

  return (
    <div className="msgr">
      <Rail
        onProfile={() => setProfile(true)}
        onNotifications={() => setNotifications(true)}
        onFeedback={() => setFeedback(true)}
      />

      <ChatList
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
            avatarName: other.displayName,
            avatarUrl: other.avatarUrl,
            online: chatOnline,
            offlineLabel,
            otherLastReadAt: activeChat.otherLastReadAt,
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

      <NewChatModal open={newChat} onClose={() => setNewChat(false)} onOpened={selectChat} />
      <NewGroupModal open={newGroup} onClose={() => setNewGroup(false)} onCreated={selectGroup} />
      <ProfileModal open={profile} onClose={() => setProfile(false)} />
      <NotificationsModal open={notifications} onClose={() => setNotifications(false)} />
      <FeedbackModal open={feedback} onClose={() => setFeedback(false)} />
    </div>
  );
}
