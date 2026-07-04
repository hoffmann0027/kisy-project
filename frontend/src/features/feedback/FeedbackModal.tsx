import { useState } from "react";
import { Avatar, Button, Modal, Spinner, toast } from "@shared/ui";
import { roleLabel } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";
import { useAuthStore } from "@shared/store/auth";
import {
  flattenFeedback,
  useCreateFeedback,
  useDeleteFeedback,
  useFeedback,
} from "@entities/feedback/queries";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function FeedbackModal({ open, onClose }: Props) {
  const me = useAuthStore((s) => s.user!);
  const isCEO = me.roleLevel === 1;
  const { data, isPending, hasNextPage, isFetchingNextPage, fetchNextPage } = useFeedback(open);
  const create = useCreateFeedback();
  const del = useDeleteFeedback();
  const [body, setBody] = useState("");

  const items = flattenFeedback(data?.pages);

  const submit = () => {
    const text = body.trim();
    if (!text) return;
    create.mutate(text, {
      onSuccess: () => {
        setBody("");
        toast.success("Спасибо за отзыв!");
      },
      onError: () => toast.error("Не удалось отправить отзыв"),
    });
  };

  const remove = (id: string) => {
    if (!window.confirm("Удалить этот отзыв?")) return;
    del.mutate(id, { onError: () => toast.error("Не удалось удалить") });
  };

  return (
    <Modal open={open} title="Отзывы и предложения" onClose={onClose}>
      <div className="feedback">
        <div className="feedback__compose">
          <textarea
            className="ui-input feedback__input"
            placeholder="Поделитесь идеей или сообщите о проблеме…"
            rows={3}
            maxLength={2000}
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
          <Button variant="primary" onClick={submit} loading={create.isPending} disabled={!body.trim()}>
            Отправить
          </Button>
        </div>

        <div className="feedback__list">
          {isPending && (
            <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
              <Spinner />
            </div>
          )}
          {!isPending && items.length === 0 && (
            <div className="feedback__empty">Пока нет отзывов. Будьте первым!</div>
          )}
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
              </div>
              {isCEO && (
                <button className="feedback__delete" title="Удалить" onClick={() => remove(f.id)}>
                  ✕
                </button>
              )}
            </div>
          ))}
          {hasNextPage && (
            <Button variant="ghost" loading={isFetchingNextPage} onClick={() => void fetchNextPage()}>
              Показать ещё
            </Button>
          )}
        </div>
      </div>
    </Modal>
  );
}
