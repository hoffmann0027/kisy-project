import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import "./board.css";
import { useCardDrag } from "./useCardDrag";
import { cn } from "@shared/lib/cn";
import { Avatar, Button, IconButton, Spinner, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { ApiError } from "@shared/api/envelope";
import { CARD_LABELS, type Board, type BoardCard, type BoardColumn, type Group, type User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useBoard, useBoardMutations, boardKeys } from "@entities/board/queries";
import { useGroupMembers } from "@entities/group/queries";
import { CardModal } from "./CardModal";

interface Props {
  group: Group;
}

export function BoardView({ group }: Props) {
  const me = useAuthStore((s) => s.user!);
  const { data: board, isPending, error } = useBoard(group.id);
  const { data: members } = useGroupMembers(group.id);
  const m = useBoardMutations(group.id);
  const qc = useQueryClient();

  const [editCard, setEditCard] = useState<BoardCard | null>(null);

  // Pointer-based dragging: the same code path serves mouse and touch (native
  // HTML5 drag events never fire on mobile browsers). Called above the early
  // returns below so the hook order stays stable; the callback may be a fresh
  // closure each render — useCardDrag re-reads it through a ref.
  const { drag, begin, consumeClick } = useCardDrag((cardId, columnId, index) => {
    // Optimistic reorder in the cache, then persist.
    qc.setQueryData<Board>(boardKeys.board(group.id), (prev) =>
      prev ? moveInBoard(prev, cardId, columnId, index) : prev,
    );
    m.moveCard.mutate({ cardId, columnId, index }, { onError: () => toast.error("Не удалось переместить задачу") });
  });

  // The group founder owns the board structure (create board, add/remove
  // columns). Known from the group DTO even before a board exists.
  const founder = group.createdBy === me.id;
  const memberList: User[] = members?.map((m) => m.user) ?? [];

  if (isPending) {
    return (
      <div className="board" style={{ alignItems: "center", justifyContent: "center", display: "flex" }}>
        <Spinner size={28} />
      </div>
    );
  }

  // No board yet.
  if (error instanceof ApiError && error.status === 404) {
    return (
      <div className="board">
        <div className="board__empty">
          <div style={{ opacity: 0.4 }}>
            <Icon.Board size={56} />
          </div>
          <div style={{ fontSize: 17, fontWeight: 600, color: "var(--color-text-primary)" }}>Доска задач</div>
          {founder ? (
            <>
              <p style={{ maxWidth: 320 }}>Создайте Kanban-доску задач для этой группы.</p>
              <Button
                loading={m.createBoard.isPending}
                onClick={() =>
                  m.createBoard.mutate("Доска задач", {
                    onError: () => toast.error("Не удалось создать доску"),
                  })
                }
              >
                Создать доску
              </Button>
            </>
          ) : (
            <p style={{ maxWidth: 320 }}>Основатель группы ещё не создал доску задач.</p>
          )}
        </div>
      </div>
    );
  }

  if (!board) {
    return <div className="board__empty">Не удалось загрузить доску</div>;
  }

  const assigneeOf = (id: string | null) => (id ? memberList.find((u) => u.id === id) : undefined);
  const draggedCard = drag ? board.columns.flatMap((c) => c.cards).find((c) => c.id === drag.cardId) : undefined;

  return (
    <div className="board">
      <div className="board__scroll" data-board-scroll>
        {board.columns.map((col) => (
          <ColumnView
            key={col.id}
            column={col}
            founder={founder}
            dropActive={drag?.overColumnId === col.id}
            draggingId={drag?.cardId ?? null}
            onCardPointerDown={(e, cardId) => begin(e, cardId, col.id)}
            onOpenCard={(card) => {
              if (consumeClick()) return; // that click ended a drag
              setEditCard(card);
            }}
            assigneeOf={assigneeOf}
            onAddCard={(title) => m.createCard.mutate({ columnId: col.id, input: { title } })}
            onRename={(title) => m.renameColumn.mutate({ columnId: col.id, title })}
            onDelete={() => m.deleteColumn.mutate(col.id)}
          />
        ))}

        {founder && (
          <div className="board-add-col">
            <AddColumn onAdd={(title) => m.addColumn.mutate({ boardId: board.id, title })} />
          </div>
        )}
      </div>

      {/* The card following the pointer. pointer-events:none keeps it out of
          the way of the hit-test under the cursor. */}
      {drag && draggedCard && (
        <div className="board-card board-card--ghost" style={{ left: drag.x, top: drag.y, width: drag.width }}>
          {draggedCard.label && CARD_LABELS[draggedCard.label] && (
            <div className="board-card__label" style={{ background: CARD_LABELS[draggedCard.label] }} />
          )}
          <div className="board-card__title">{draggedCard.title}</div>
        </div>
      )}

      <CardModal
        key={editCard?.id ?? "none"}
        card={editCard}
        members={memberList}
        canDelete={!!editCard && (editCard.createdBy === me.id || founder)}
        onClose={() => setEditCard(null)}
        onSave={(cardId, input) => m.updateCard.mutate({ cardId, input })}
        onDelete={(cardId) => m.deleteCard.mutate(cardId)}
      />
    </div>
  );
}

interface ColumnProps {
  column: BoardColumn;
  founder: boolean;
  dropActive: boolean;
  draggingId: string | null;
  onCardPointerDown: (e: React.PointerEvent<HTMLElement>, cardId: string) => void;
  onOpenCard: (card: BoardCard) => void;
  assigneeOf: (id: string | null) => User | undefined;
  onAddCard: (title: string) => void;
  onRename: (title: string) => void;
  onDelete: () => void;
}

function ColumnView(p: ColumnProps) {
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const [renaming, setRenaming] = useState(false);
  const [title, setTitle] = useState(p.column.title);

  const submitCard = () => {
    if (draft.trim()) p.onAddCard(draft.trim());
    setDraft("");
    setAdding(false);
  };

  return (
    <div className={cn("board-col", p.dropActive && "board-col--drop")} data-column-id={p.column.id}>
      <div className="board-col__header">
        {renaming ? (
          <input
            className="board-col__title-input"
            value={title}
            autoFocus
            onChange={(e) => setTitle(e.target.value)}
            onBlur={() => {
              setRenaming(false);
              if (title.trim() && title !== p.column.title) p.onRename(title.trim());
            }}
            onKeyDown={(e) => e.key === "Enter" && (e.target as HTMLInputElement).blur()}
          />
        ) : (
          <>
            <span style={{ flex: 1 }} onDoubleClick={() => p.founder && setRenaming(true)}>
              {p.column.title}
            </span>
            <span className="board-col__count">{p.column.cards.length}</span>
            {p.founder && (
              <IconButton label="Удалить колонку" onClick={p.onDelete}>
                <Icon.Trash size={15} />
              </IconButton>
            )}
          </>
        )}
      </div>

      <div className="board-col__body" data-column-body>
        {p.column.cards.map((card) => (
          <CardView
            key={card.id}
            card={card}
            assignee={p.assigneeOf(card.assigneeId)}
            dragging={p.draggingId === card.id}
            onPointerDown={(e) => p.onCardPointerDown(e, card.id)}
            onOpen={() => p.onOpenCard(card)}
          />
        ))}
      </div>

      <div className="board-col__add">
        {adding ? (
          <>
            <textarea
              className="board-col__add-input"
              rows={2}
              autoFocus
              value={draft}
              placeholder="Название задачи…"
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  submitCard();
                }
              }}
            />
            <div style={{ display: "flex", gap: 6, marginTop: 6 }}>
              <Button onClick={submitCard}>Добавить</Button>
              <Button variant="ghost" onClick={() => setAdding(false)}>
                Отмена
              </Button>
            </div>
          </>
        ) : (
          <button className="board-col__add-btn" onClick={() => setAdding(true)}>
            + Добавить задачу
          </button>
        )}
      </div>
    </div>
  );
}

function CardView({
  card,
  assignee,
  dragging,
  onPointerDown,
  onOpen,
}: {
  card: BoardCard;
  assignee?: User;
  dragging: boolean;
  onPointerDown: (e: React.PointerEvent<HTMLElement>) => void;
  onOpen: () => void;
}) {
  const due = card.dueDate ? new Date(card.dueDate) : null;
  const dueClass = due ? (due.getTime() < Date.now() ? "board-card__due--overdue" : due.getTime() - Date.now() < 3 * 86400000 ? "board-card__due--soon" : "") : "";

  return (
    <div
      className={cn("board-card", dragging && "board-card--dragging")}
      data-card-id={card.id}
      onPointerDown={onPointerDown}
      onClick={onOpen}
    >
      {card.label && CARD_LABELS[card.label] && (
        <div className="board-card__label" style={{ background: CARD_LABELS[card.label] }} />
      )}
      <div className="board-card__title">{card.title}</div>
      <div className="board-card__footer">
        {due ? (
          <span className={cn("board-card__due", dueClass)}>
            <Icon.Calendar size={13} />
            {due.toLocaleDateString("ru-RU", { day: "numeric", month: "short" })}
          </span>
        ) : (
          <span />
        )}
        {assignee && <Avatar name={assignee.displayName} url={assignee.avatarUrl} size={24} />}
      </div>
    </div>
  );
}

function AddColumn({ onAdd }: { onAdd: (title: string) => void }) {
  const [adding, setAdding] = useState(false);
  const [title, setTitle] = useState("");
  if (!adding) {
    return <button onClick={() => setAdding(true)}>+ Добавить колонку</button>;
  }
  return (
    <div className="board-col" style={{ padding: 10 }}>
      <input
        className="board-col__add-input"
        autoFocus
        value={title}
        placeholder="Название колонки"
        onChange={(e) => setTitle(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && title.trim()) {
            onAdd(title.trim());
            setTitle("");
            setAdding(false);
          }
        }}
      />
      <div style={{ display: "flex", gap: 6, marginTop: 6 }}>
        <Button
          onClick={() => {
            if (title.trim()) onAdd(title.trim());
            setTitle("");
            setAdding(false);
          }}
        >
          Добавить
        </Button>
        <Button variant="ghost" onClick={() => setAdding(false)}>
          Отмена
        </Button>
      </div>
    </div>
  );
}

// moveInBoard returns a new board with the card relocated (optimistic UI).
function moveInBoard(board: Board, cardId: string, toColumnId: string, toIndex: number): Board {
  let moved: BoardCard | undefined;
  const stripped = board.columns.map((col) => {
    const idx = col.cards.findIndex((c) => c.id === cardId);
    if (idx === -1) return col;
    moved = col.cards[idx];
    return { ...col, cards: col.cards.filter((c) => c.id !== cardId) };
  });
  if (!moved) return board;
  const movedCard: BoardCard = { ...moved, columnId: toColumnId };
  return {
    ...board,
    columns: stripped.map((col) => {
      if (col.id !== toColumnId) return col;
      const cards = col.cards.slice();
      cards.splice(Math.min(toIndex, cards.length), 0, movedCard);
      return { ...col, cards };
    }),
  };
}
