// Thread panel (UPD3 stage K): a right-hand overlay showing one root
// message and its replies, with the shared Composer posting into the
// thread (threadRootId). Realtime updates arrive via the WS layer, which
// routes thread replies into this panel's query cache.
import { useEffect, useMemo, useRef, useState } from "react";
import { Button, Spinner, toast } from "@shared/ui";
import type { Attachment, Message } from "@shared/api/types";
import {
  flattenMessages,
  useMessageCacheWriter,
  useSendMessage,
  useThreadMessages,
} from "@entities/message/queries";
import { useAuthStore } from "@shared/store/auth";
import { MessageBubble } from "@widgets/conversation/MessageBubble";
import { Composer } from "@widgets/conversation/Composer";

interface Props {
  root: Message;
  onClose: () => void;
  /** Delegated bubble handlers from the parent conversation. */
  onReact: (m: Message, emoji: string) => void;
  onDelete: (m: Message) => void;
  onEdit: (m: Message, text: string) => void;
  onOpenImage: (attachment: Attachment) => void;
}

export function ThreadPanel({ root, onClose, onReact, onDelete, onEdit, onOpenImage }: Props) {
  const me = useAuthStore((s) => s.user!);
  const { data, isPending, fetchNextPage, hasNextPage, isFetchingNextPage } = useThreadMessages(root.id);
  const send = useSendMessage("group", root.chatId);
  const cache = useMessageCacheWriter();
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const replies = useMemo(() => flattenMessages(data?.pages), [data]);

  const lastCount = useRef(0);
  useEffect(() => {
    if (replies.length > lastCount.current) {
      bottomRef.current?.scrollIntoView({ behavior: lastCount.current === 0 ? "auto" : "smooth" });
    }
    lastCount.current = replies.length;
  }, [replies.length]);

  const handleSend = (text: string, replyToId?: string, attachments?: Attachment[]) => {
    send.mutate(
      { text, replyTo: replyToId, attachmentIds: attachments?.map((a) => a.id), threadRootId: root.id },
      {
        onSuccess: ({ message }) => cache.insertThread(root.id, message),
        onError: () => toast.error("Не удалось отправить в обсуждение"),
      },
    );
  };

  const previewFor = (id: string | null): string | undefined => {
    if (!id) return undefined;
    const parent = id === root.id ? root : replies.find((m) => m.id === id);
    if (!parent) return undefined;
    return parent.isDeleted ? "удалённое сообщение" : (parent.text ?? "").slice(0, 80);
  };

  const noop = () => {};

  const bubble = (m: Message) => (
    <MessageBubble
      key={m.id}
      message={m}
      mine={m.senderId === me.id}
      canDelete={m.senderId === me.id || me.roleLevel === 1}
      canEdit={m.senderId === me.id && !m.pending && !m.failed && !m.encrypted}
      replyPreview={previewFor(m.replyTo)}
      onReply={setReplyTo}
      onEdit={onEdit}
      onDelete={onDelete}
      onReact={onReact}
      onPin={noop}
      onOpenImage={onOpenImage}
      onForward={noop}
    />
  );

  return (
    <aside className="cpanel thread-panel" aria-label="Обсуждение">
      <header className="cpanel__header">
        <span className="cpanel__title">Обсуждение</span>
        <button className="cpanel__close" onClick={onClose} title="Закрыть обсуждение">
          ✕
        </button>
      </header>

      <div className="thread-panel__scroll">
        <div className="thread-panel__root">{bubble(root)}</div>
        <div className="thread-panel__divider">
          {root.threadReplyCount
            ? `Ответы: ${root.threadReplyCount}`
            : "Пока нет ответов — начните обсуждение"}
        </div>
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 16 }}>
            <Spinner />
          </div>
        )}
        {hasNextPage && (
          <div className="conv__load-more">
            <Button variant="ghost" loading={isFetchingNextPage} onClick={() => void fetchNextPage()}>
              Загрузить ещё
            </Button>
          </div>
        )}
        {replies.map(bubble)}
        <div ref={bottomRef} />
      </div>

      <Composer
        chatType="group"
        chatId={root.chatId}
        draftKey={`${root.chatId}:thread:${root.id}`}
        replyTo={replyTo}
        replyPreview={previewFor(replyTo?.id ?? null)}
        onClearReply={() => setReplyTo(null)}
        onSend={handleSend}
      />
    </aside>
  );
}
