import { useState } from "react";
import { Avatar, Button, Modal, Spinner, toast } from "@shared/ui";
import { roleLabel, type FeedbackItem, type FeedbackScope } from "@shared/api/types";
import { userFacingError } from "@shared/api/envelope";
import { formatRelative } from "@shared/lib/format";
import { useAuthStore } from "@shared/store/auth";
import { t } from "@shared/i18n";
import {
  canAnswerFeedback,
  feedbackWait,
  flattenFeedback,
  useCreateFeedback,
  useDeleteFeedback,
  useFeedback,
  useReplyFeedback,
} from "@entities/feedback/queries";

// "Отзывы и предложения" is a private line to leadership: everyone sees only
// their own entries, with the answer once there is one, and may write once a
// day. Levels 1-3 also get the inbox — every entry nobody has answered yet —
// and answer from it; an answered entry leaves the inbox. The server enforces
// all of it; this screen only follows.

interface Props {
  open: boolean;
  onClose: () => void;
}

export function FeedbackModal({ open, onClose }: Props) {
  const me = useAuthStore((s) => s.user!);
  const staff = canAnswerFeedback(me.roleLevel);
  const [tab, setTab] = useState<FeedbackScope>(staff ? "inbox" : "mine");
  const scope: FeedbackScope = staff ? tab : "mine";

  return (
    <Modal open={open} title={t("hub.feedback.title")} onClose={onClose}>
      <div className="feedback">
        {staff && (
          <div className="ui-tabs" role="tablist">
            <button role="tab" aria-selected={scope === "inbox"} className={scope === "inbox" ? "is-active" : ""} onClick={() => setTab("inbox")}>
              {t("hub.feedback.inbox")}
            </button>
            <button role="tab" aria-selected={scope === "mine"} className={scope === "mine" ? "is-active" : ""} onClick={() => setTab("mine")}>
              {t("hub.feedback.mine")}
            </button>
          </div>
        )}
        {scope === "inbox" ? (
          <Inbox enabled={open} isCEO={me.roleLevel === 1} />
        ) : (
          <Mine enabled={open} isCEO={me.roleLevel === 1} />
        )}
      </div>
    </Modal>
  );
}

function waitText(ms: number): string {
  const hours = Math.ceil(ms / 3_600_000);
  return hours <= 1 ? t("hub.feedback.waitUnderHour") : t("hub.feedback.waitHours", { count: hours });
}

function Mine({ enabled, isCEO }: { enabled: boolean; isCEO: boolean }) {
  const { data, isPending, hasNextPage, isFetchingNextPage, fetchNextPage } = useFeedback("mine", enabled);
  const create = useCreateFeedback();
  const [body, setBody] = useState("");
  const items = flattenFeedback(data?.pages);
  const wait = isPending ? 0 : feedbackWait(items[0]?.createdAt);

  const submit = () => {
    const text = body.trim();
    if (!text) return;
    create.mutate(text, {
      onSuccess: () => {
        setBody("");
        toast.success(t("hub.feedback.sent"));
      },
      onError: (e) => toast.error(userFacingError(e, t("hub.feedback.sendFailed"))),
    });
  };

  return (
    <>
      <div className="feedback__compose">
        <textarea
          className="ui-input feedback__input"
          placeholder={t("hub.feedback.placeholder")}
          aria-label={t("hub.feedback.inputLabel")}
          rows={3}
          maxLength={2000}
          value={body}
          disabled={wait > 0}
          onChange={(e) => setBody(e.target.value)}
        />
        <Button variant="primary" onClick={submit} loading={create.isPending} disabled={!body.trim() || wait > 0}>
          {t("hub.feedback.send")}
        </Button>
        <p className="feedback__hint">
          {wait > 0
            ? t("hub.feedback.waitHint", { when: waitText(wait) })
            : t("hub.feedback.privacyHint")}
        </p>
      </div>
      <FeedbackList
        items={items}
        isPending={isPending}
        empty={t("hub.feedback.mineEmpty")}
        renderExtra={(f) =>
          f.reply ? (
            <div className="feedback__reply">
              <div className="feedback__reply-who">
                {t("hub.feedback.replyHeading")}
                {f.reply.by ? ` · ${f.reply.by.displayName}` : ""}
                {f.reply.by?.roleLevel ? ` · ${roleLabel(f.reply.by.roleLevel)}` : ""} · {formatRelative(f.reply.at)}
              </div>
              <div className="feedback__text">{f.reply.body}</div>
            </div>
          ) : (
            <div className="feedback__waiting">{t("hub.feedback.awaiting")}</div>
          )
        }
        isCEO={isCEO}
      />
      {hasNextPage && (
        <Button variant="ghost" loading={isFetchingNextPage} onClick={() => void fetchNextPage()}>
          {t("hub.feedback.more")}
        </Button>
      )}
    </>
  );
}

function Inbox({ enabled, isCEO }: { enabled: boolean; isCEO: boolean }) {
  const { data, isPending, hasNextPage, isFetchingNextPage, fetchNextPage } = useFeedback("inbox", enabled);
  const items = flattenFeedback(data?.pages);
  return (
    <>
      <FeedbackList
        items={items}
        isPending={isPending}
        empty={t("hub.feedback.inboxEmpty")}
        renderExtra={(f) => <AnswerBox id={f.id} />}
        isCEO={isCEO}
      />
      {hasNextPage && (
        <Button variant="ghost" loading={isFetchingNextPage} onClick={() => void fetchNextPage()}>
          {t("hub.feedback.more")}
        </Button>
      )}
    </>
  );
}

function AnswerBox({ id }: { id: string }) {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const reply = useReplyFeedback();

  if (!open) {
    return (
      <div className="feedback__answer-actions" style={{ marginTop: 8 }}>
        <Button variant="secondary" onClick={() => setOpen(true)}>
          {t("hub.feedback.answer")}
        </Button>
      </div>
    );
  }
  const send = () => {
    const body = text.trim();
    if (!body) return;
    reply.mutate(
      { id, body },
      {
        // The entry leaves the inbox with the refetch.
        onSuccess: () => toast.success(t("hub.feedback.replySent")),
        onError: (e) => toast.error(userFacingError(e, t("hub.feedback.replyFailed"))),
      },
    );
  };
  return (
    <div className="feedback__answer">
      <textarea
        className="ui-input feedback__input"
        aria-label={t("hub.feedback.replyInputLabel")}
        placeholder={t("hub.feedback.replyPlaceholder")}
        rows={3}
        maxLength={2000}
        autoFocus
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <div className="feedback__answer-actions">
        <Button variant="secondary" onClick={() => setOpen(false)} disabled={reply.isPending}>
          {t("hub.feedback.cancel")}
        </Button>
        <Button onClick={send} loading={reply.isPending} disabled={!text.trim()}>
          {t("hub.feedback.sendReply")}
        </Button>
      </div>
    </div>
  );
}

function FeedbackList({
  items,
  isPending,
  empty,
  renderExtra,
  isCEO,
}: {
  items: FeedbackItem[];
  isPending: boolean;
  empty: string;
  renderExtra: (f: FeedbackItem) => React.ReactNode;
  isCEO: boolean;
}) {
  const del = useDeleteFeedback();
  const remove = (id: string) => {
    if (!window.confirm(t("hub.feedback.deleteConfirm"))) return;
    del.mutate(id, { onError: () => toast.error(t("hub.feedback.deleteFailed")) });
  };
  return (
    <div className="feedback__list">
      {isPending && (
        <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
          <Spinner />
        </div>
      )}
      {!isPending && items.length === 0 && <div className="feedback__empty">{empty}</div>}
      {items.map((f) => (
        <div key={f.id} className="feedback__item">
          <Avatar name={f.author.displayName} url={f.author.avatarUrl} size={38} />
          <div className="feedback__body">
            <div className="feedback__meta">
              <span className="feedback__author">{f.author.displayName}</span>
              <span className="feedback__role">{roleLabel(f.author.roleLevel)}</span>
              <span className="feedback__date">{formatRelative(f.createdAt)}</span>
            </div>
            <div className="feedback__text">{f.body}</div>
            {renderExtra(f)}
          </div>
          {isCEO && (
            <button className="feedback__delete" title={t("hub.feedback.delete")} onClick={() => remove(f.id)}>
              ✕
            </button>
          )}
        </div>
      ))}
    </div>
  );
}
