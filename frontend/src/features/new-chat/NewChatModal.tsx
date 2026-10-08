import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { t } from "@shared/i18n";
import { Avatar, Modal, Spinner, toast } from "@shared/ui";
import { usersApi } from "@shared/api/endpoints";
import { userSubtitle, type Chat } from "@shared/api/types";
import { useOpenChat } from "@entities/chat/queries";
import { ApiError } from "@shared/api/envelope";
import { VerifiedName } from "@shared/ui/VerifiedBadge";

interface Props {
  open: boolean;
  onClose: () => void;
  onOpened: (chat: Chat) => void;
}

export function NewChatModal({ open, onClose, onOpened }: Props) {
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const openChat = useOpenChat();

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query), 250);
    return () => clearTimeout(timer);
  }, [query]);

  const { data, isPending } = useQuery({
    queryKey: ["directory", debounced],
    queryFn: async () => (await usersApi.directory(debounced)).users,
    enabled: open,
  });

  const select = (userId: string) => {
    openChat.mutate(userId, {
      onSuccess: (chat) => {
        onOpened(chat);
        onClose();
        setQuery("");
      },
      onError: (e) => {
        toast.error(
          e instanceof ApiError && e.status === 403
            ? t("chat.newChat.forbidden")
            : t("chat.newChat.openFailed"),
        );
      },
    });
  };

  return (
    <Modal open={open} title={t("chat.list.newChat")} onClose={onClose}>
      <input
        className="ui-input"
        placeholder={t("chat.newChat.searchPlaceholder")}
        autoFocus
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div style={{ maxHeight: 340, overflowY: "auto", display: "flex", flexDirection: "column", gap: 2 }}>
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
            <Spinner />
          </div>
        )}
        {!isPending && (data?.length ?? 0) === 0 && (
          <div style={{ textAlign: "center", color: "var(--color-text-secondary)", padding: 20, fontSize: 14 }}>
            {t("chat.newChat.noUsers")}
          </div>
        )}
        {data?.map((u) => (
          <button key={u.id} className="user-row" onClick={() => select(u.id)} disabled={openChat.isPending}>
            <Avatar name={u.displayName} url={u.avatarUrl} size={40} />
            <div>
              <div className="user-row__name">
                <VerifiedName name={u.displayName} verified={!!u.verifiedAt} />
              </div>
              <div className="user-row__role">
                {userSubtitle(u)}
              </div>
            </div>
          </button>
        ))}
      </div>
    </Modal>
  );
}
