// Context menu of a chat-list row (UPD3 stage H): archive/unarchive the
// chat and toggle its membership in the user's folders. Opens at the cursor
// (right-click / long-press friendly), closes on outside click or Escape.
import { useEffect, useRef } from "react";
import { Icon } from "@shared/ui/icons";
import { toast } from "@shared/ui";
import type { ChatType } from "@shared/api/types";
import {
  chatKey,
  isArchived,
  useArchiveChat,
  useArchived,
  useFolderItem,
  useFolders,
} from "@entities/chat-folders/queries";

export interface MenuTarget {
  chatType: ChatType;
  chatId: string;
  x: number;
  y: number;
}

interface Props {
  target: MenuTarget;
  onClose: () => void;
}

export function ChatContextMenu({ target, onClose }: Props) {
  const { folders } = useFolders();
  const { archivedSet } = useArchived();
  const archiveChat = useArchiveChat();
  const folderItem = useFolderItem();
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  const archived = isArchived(archivedSet, target.chatType, target.chatId);
  const key = chatKey(target.chatType, target.chatId);

  const toggleArchive = () => {
    archiveChat.mutate(
      { chatType: target.chatType, chatId: target.chatId, archive: !archived },
      {
        onSuccess: () => toast.success(archived ? "Чат возвращён из архива" : "Чат в архиве"),
        onError: () => toast.error("Не удалось изменить архив"),
      },
    );
    onClose();
  };

  const toggleFolder = (folderId: string, inFolder: boolean) => {
    folderItem.mutate(
      { folderId, chatType: target.chatType, chatId: target.chatId, add: !inFolder },
      { onError: () => toast.error("Не удалось изменить папку") },
    );
  };

  // Keep the menu inside the viewport.
  const style: React.CSSProperties = {
    left: Math.min(target.x, window.innerWidth - 240),
    top: Math.min(target.y, window.innerHeight - (120 + folders.length * 36)),
  };

  return (
    <div className="chatmenu" style={style} ref={rootRef} role="menu">
      <button className="chatmenu__item" onClick={toggleArchive}>
        <Icon.Archive size={16} />
        {archived ? "Вернуть из архива" : "В архив"}
      </button>
      {folders.length > 0 && <div className="chatmenu__divider" />}
      {folders.map((f) => {
        const inFolder = f.items.some((i) => chatKey(i.chatType, i.chatId) === key);
        return (
          <button key={f.id} className="chatmenu__item" onClick={() => toggleFolder(f.id, inFolder)}>
            <Icon.Folder size={16} />
            <span className="chatmenu__label">{f.name}</span>
            {inFolder && <Icon.Check size={16} />}
          </button>
        );
      })}
    </div>
  );
}
