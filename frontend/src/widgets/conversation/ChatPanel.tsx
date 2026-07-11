// Right-hand context panel (stage C): the chat's shared Media / Files /
// Links, lazily loaded per tab, clearance-scoped by the backend. Clicking a
// thumbnail opens the media viewer over the tab's items.
import { useState } from "react";
import { cn } from "@shared/lib/cn";
import { Button } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { formatBytes, fileTypeLabel } from "@entities/attachment/upload";
import { useChatLinks, useChatMedia } from "@entities/chatmedia/queries";
import type { ChatMediaItem, ChatType } from "@shared/api/types";

type Tab = "media" | "files" | "links";

interface Props {
  chatType: ChatType;
  chatId: string;
  onClose: () => void;
  onOpenMedia: (items: ChatMediaItem[], index: number) => void;
}

function Skeletons({ grid }: { grid?: boolean }) {
  return (
    <div className={grid ? "cpanel__grid" : "cpanel__list"}>
      {Array.from({ length: grid ? 9 : 5 }, (_, i) => (
        <div key={i} className={cn("cpanel__skeleton", grid && "cpanel__skeleton--tile")} />
      ))}
    </div>
  );
}

export function ChatPanel({ chatType, chatId, onClose, onOpenMedia }: Props) {
  const [tab, setTab] = useState<Tab>("media");

  const media = useChatMedia(chatType, chatId, "media", tab === "media");
  const files = useChatMedia(chatType, chatId, "files", tab === "files");
  const links = useChatLinks(chatType, chatId, tab === "links");

  const mediaItems = media.data?.pages.flatMap((p) => p.items) ?? [];
  const fileItems = files.data?.pages.flatMap((p) => p.items) ?? [];
  const linkItems = links.data?.pages.flatMap((p) => p.items) ?? [];

  const active = tab === "media" ? media : tab === "files" ? files : links;

  return (
    <aside className="cpanel" aria-label="Материалы чата">
      <header className="cpanel__header">
        <span className="cpanel__title">Материалы</span>
        <button className="cpanel__close" onClick={onClose} title="Закрыть панель">
          ✕
        </button>
      </header>
      <div className="cpanel__tabs" role="tablist">
        {(
          [
            ["media", "Медиа"],
            ["files", "Файлы"],
            ["links", "Ссылки"],
          ] as [Tab, string][]
        ).map(([key, label]) => (
          <button
            key={key}
            role="tab"
            aria-selected={tab === key}
            className={cn("cpanel__tab", tab === key && "cpanel__tab--active")}
            onClick={() => setTab(key)}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="cpanel__body">
        {active.isPending ? (
          <Skeletons grid={tab === "media"} />
        ) : tab === "media" ? (
          mediaItems.length === 0 ? (
            <div className="cpanel__empty">В этом чате пока нет медиа</div>
          ) : (
            <div className="cpanel__grid">
              {mediaItems.map((it, i) => (
                <button
                  key={it.attachment.id}
                  className="cpanel__tile"
                  onClick={() => onOpenMedia(mediaItems, i)}
                  title={it.attachment.fileName}
                >
                  <img src={it.attachment.url} alt={it.attachment.fileName} loading="lazy" />
                </button>
              ))}
            </div>
          )
        ) : tab === "files" ? (
          fileItems.length === 0 ? (
            <div className="cpanel__empty">В этом чате пока нет файлов</div>
          ) : (
            <div className="cpanel__list">
              {fileItems.map((it) => (
                <a
                  key={it.attachment.id}
                  className="cpanel__file"
                  href={it.attachment.url}
                  download={it.attachment.fileName}
                >
                  <span className="cpanel__file-type">{fileTypeLabel(it.attachment.fileName)}</span>
                  <span className="cpanel__file-body">
                    <span className="cpanel__file-name">{it.attachment.fileName}</span>
                    <span className="cpanel__file-size">{formatBytes(it.attachment.sizeBytes)}</span>
                  </span>
                  <Icon.Paperclip size={14} />
                </a>
              ))}
            </div>
          )
        ) : linkItems.length === 0 ? (
          <div className="cpanel__empty">
            В этом чате пока нет ссылок
            {chatType === "private" && (
              <div className="cpanel__hint">Ссылки из зашифрованных сообщений серверу не видны</div>
            )}
          </div>
        ) : (
          <div className="cpanel__list">
            {linkItems.map((it, i) => (
              <a
                key={`${it.messageId}-${i}`}
                className="cpanel__link"
                href={it.url}
                target="_blank"
                rel="noopener noreferrer"
              >
                {it.url}
              </a>
            ))}
          </div>
        )}

        {active.hasNextPage && (
          <div className="cpanel__more">
            <Button variant="ghost" loading={active.isFetchingNextPage} onClick={() => void active.fetchNextPage()}>
              Загрузить ещё
            </Button>
          </div>
        )}
      </div>
    </aside>
  );
}
