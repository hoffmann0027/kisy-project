import { useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import "./board.css";
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
  const dragCard = useRef<{ id: string; from: string } | null>(null);
  const [dropCol, setDropCol] = useState<string | null>(null);

  // The group founder owns the board structure (create board, add/remove
  // columns). Known from the group DTO even before a board exists.
  const founder = group.createdBy === me.id;
  const memberList: User[] = members ?? [];

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

  // --- drag & drop ---
  const onDrop = (columnId: string, index: number) => {
    const drag = dragCard.current;
    setDropCol(null);
    dragCard.current = null;
    if (!drag) return;

    // Optimistic reorder in the cache, then persist.
    qc.setQueryData<Board>(boardKeys.board(group.id), (prev) => (prev ? moveInBoard(prev, drag.id, columnId, index) : prev));
    m.moveCard.mutate(
      { cardId: drag.id, columnId, index },
      { onError: () => toast.error("Не удалось переместить задачу") },
    );
  };

  return (
    <div className="board">
      <div className="board__scroll">
        {board.columns.map((col) => (
          <ColumnView
            key={col.id}
            column={col}
            founder={founder}
            dropActive={dropCol === col.id}
            onDragEnterCol={() => setDropCol(col.id)}
            onDropAtEnd={() => onDrop(col.id, col.cards.length)}
            onDropBefore={(index) => onDrop(col.id, index)}
            onDragStartCard={(cardId) => (dragCard.current = { id: cardId, from: col.id })}
            onOpenCard={setEditCard}
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
  onDragEnterCol: () => void;
  onDropAtEnd: () => void;
  onDropBefore: (index: number) => void;
  onDragStartCard: (cardId: string) => void;
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
    <div
      className={cn("board-col", p.dropActive && "board-col--drop")}
      onDragOver={(e) => {
        e.preventDefault();
        p.onDragEnterCol();
      }}
      onDrop={(e) => {
        e.preventDefault();
        p.onDropAtEnd();
      }}
    >
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

      <div className="board-col__body">
        {p.column.cards.map((card, i) => (
          <div
            key={card.id}
            onDrop={(e) => {
              e.preventDefault();
              e.stopPropagation();
              p.onDropBefore(i);
            }}
            onDragOver={(e) => {
              e.preventDefault();
              p.onDragEnterCol();
            }}
          >
            <CardView card={card} assignee={p.assigneeOf(card.assigneeId)} onOpen={() => p.onOpenCard(card)} onDragStart={() => p.onDragStartCard(card.id)} />
          </div>
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

function CardView({ card, assignee, onOpen, onDragStart }: { card: BoardCard; assignee?: User; onOpen: () => void; onDragStart: () => void }) {
  const [dragging, setDragging] = useState(false);
  const due = card.dueDate ? new Date(card.dueDate) : null;
  const dueClass = due ? (due.getTime() < Date.now() ? "board-card__due--overdue" : due.getTime() - Date.now() < 3 * 86400000 ? "board-card__due--soon" : "") : "";

  return (
    <div
      className={cn("board-card", dragging && "board-card--dragging")}
      draggable
      onDragStart={() => {
        setDragging(true);
        onDragStart();
      }}
      onDragEnd={() => setDragging(false)}
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
