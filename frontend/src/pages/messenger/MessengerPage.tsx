import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import "./messenger.css";
import { Rail } from "@widgets/rail/Rail";
import { ChatList } from "@widgets/chat-list/ChatList";
import { Conversation } from "@widgets/conversation/Conversation";
import { NewChatModal } from "@features/new-chat/NewChatModal";
import { ProfileModal } from "@features/profile/ProfileModal";
import { NotificationsModal } from "@features/notifications/NotificationsModal";
import { Icon } from "@shared/ui/icons";
import type { Chat } from "@shared/api/types";
import { useChats } from "@entities/chat/queries";
import { useRealtime } from "@app/useRealtime";

export function MessengerPage() {
  useRealtime();
  const navigate = useNavigate();
  const { chatId } = useParams();
  const { data: chats } = useChats();

  const [newChat, setNewChat] = useState(false);
  const [profile, setProfile] = useState(false);
  const [notifications, setNotifications] = useState(false);

  const activeChat: Chat | undefined = chats?.find((c) => c.id === chatId);

  const select = (chat: Chat) => navigate(`/chat/${chat.id}`);

  return (
    <div className="msgr">
      <Rail onProfile={() => setProfile(true)} onNotifications={() => setNotifications(true)} />

      <ChatList activeChatId={chatId ?? null} onSelect={select} onNewChat={() => setNewChat(true)} />

      {activeChat ? (
        <Conversation key={activeChat.id} chat={activeChat} />
      ) : (
        <div className="ui-empty">
          <div style={{ opacity: 0.4 }}>
            <Icon.Chat size={64} />
          </div>
          <div style={{ fontSize: 17, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Выберите чат
          </div>
          <div>Откройте существующий диалог или начните новый</div>
        </div>
      )}

      <NewChatModal open={newChat} onClose={() => setNewChat(false)} onOpened={select} />
      <ProfileModal open={profile} onClose={() => setProfile(false)} />
      <NotificationsModal open={notifications} onClose={() => setNotifications(false)} />
    </div>
  );
}
