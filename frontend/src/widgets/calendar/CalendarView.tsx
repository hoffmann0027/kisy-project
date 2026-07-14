import { useMemo, useRef, useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { useAuthStore } from "@shared/store/auth";
import {
  CALENDAR_COLORS,
  type CalendarCardRef,
  type CalendarColor,
  type CalendarEvent,
  type Group,
} from "@shared/api/types";
import { useCalendarMonth, useCreateEvent, useDeleteEvent, useUpdateEvent } from "@entities/calendar/queries";
import "./calendar.css";

// Visible hex per palette colour (theme-agnostic; chips carry their own bg).
const COLOR_HEX: Record<CalendarColor, string> = {
  blue: "#3b82f6",
  green: "#22c55e",
  red: "#ef4444",
  orange: "#f59e0b",
  purple: "#a855f7",
  teal: "#14b8a6",
  pink: "#ec4899",
  gray: "#6b7280",
};

const WEEKDAYS = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];
const MONTHS = [
  "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
  "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
];

function ymd(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

// 42-cell grid (6 weeks) starting on the Monday on/before the 1st.
function monthGrid(year: number, month: number): Date[] {
  const first = new Date(year, month, 1);
  const offset = (first.getDay() + 6) % 7; // Mon=0 … Sun=6
  const start = new Date(year, month, 1 - offset);
  return Array.from({ length: 42 }, (_, i) => new Date(start.getFullYear(), start.getMonth(), start.getDate() + i));
}

// datetime-local value ("YYYY-MM-DDTHH:mm") for a Date in local time.
function toLocalInput(d: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}
function timeLabel(iso: string): string {
  const d = new Date(iso);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

interface Props {
  group: Group;
  onOpenCard: (cardId: string) => void;
}

export function CalendarView({ group, onOpenCard }: Props) {
  const me = useAuthStore((s) => s.user!);
  const today = new Date();
  const [cursor, setCursor] = useState({ year: today.getFullYear(), month: today.getMonth() });

  const grid = useMemo(() => monthGrid(cursor.year, cursor.month), [cursor]);
  const fromISO = grid[0].toISOString();
  const toISO = new Date(grid[41].getFullYear(), grid[41].getMonth(), grid[41].getDate() + 1).toISOString();

  const { data, isPending } = useCalendarMonth(group.id, fromISO, toISO);
  const createEvent = useCreateEvent(group.id);
  const updateEvent = useUpdateEvent(group.id);
  const deleteEvent = useDeleteEvent(group.id);

  const [creating, setCreating] = useState<string | null>(null); // ymd of clicked day
  const [editing, setEditing] = useState<CalendarEvent | null>(null);

  // Group events + card refs by day (ymd).
  const byDay = useMemo(() => {
    const map = new Map<string, { events: CalendarEvent[]; cards: CalendarCardRef[] }>();
    const bucket = (key: string) => {
      let b = map.get(key);
      if (!b) map.set(key, (b = { events: [], cards: [] }));
      return b;
    };
    data?.events.forEach((e) => bucket(ymd(new Date(e.startsAt))).events.push(e));
    data?.cards.forEach((c) => bucket(ymd(new Date(c.dueDate))).cards.push(c));
    return map;
  }, [data]);

  const shift = (delta: number) => {
    const d = new Date(cursor.year, cursor.month + delta, 1);
    setCursor({ year: d.getFullYear(), month: d.getMonth() });
  };

  // Touch swipe to change months.
  const touchX = useRef<number | null>(null);
  const onTouchStart = (e: React.TouchEvent) => (touchX.current = e.touches[0].clientX);
  const onTouchEnd = (e: React.TouchEvent) => {
    if (touchX.current === null) return;
    const dx = e.changedTouches[0].clientX - touchX.current;
    if (Math.abs(dx) > 50) shift(dx < 0 ? 1 : -1);
    touchX.current = null;
  };

  const todayYmd = ymd(today);

  return (
    <div className="cal">
      <header className="cal__bar">
        <button className="cal__nav" onClick={() => shift(-1)} aria-label="Предыдущий месяц">‹</button>
        <div className="cal__title">{MONTHS[cursor.month]} {cursor.year}</div>
        <button className="cal__nav" onClick={() => shift(1)} aria-label="Следующий месяц">›</button>
      </header>

      <div className="cal__weekdays">
        {WEEKDAYS.map((w) => (
          <div key={w} className="cal__weekday">{w}</div>
        ))}
      </div>

      <div className="cal__grid" onTouchStart={onTouchStart} onTouchEnd={onTouchEnd}>
        {grid.map((day) => {
          const key = ymd(day);
          const inMonth = day.getMonth() === cursor.month;
          const b = byDay.get(key);
          return (
            <button
              key={key}
              className={`cal__cell${inMonth ? "" : " cal__cell--out"}${key === todayYmd ? " cal__cell--today" : ""}`}
              onClick={() => setCreating(key)}
            >
              <span className="cal__daynum">{day.getDate()}</span>
              <span className="cal__chips">
                {isPending && inMonth && <span className="cal__skeleton" />}
                {b?.events.map((e) => (
                  <span
                    key={e.id}
                    className="cal__chip"
                    style={{ background: COLOR_HEX[e.color] }}
                    onClick={(ev) => {
                      ev.stopPropagation();
                      setEditing(e);
                    }}
                    title={e.title}
                  >
                    {timeLabel(e.startsAt)} {e.title}
                  </span>
                ))}
                {b?.cards.map((c) => (
                  <span
                    key={c.cardId}
                    className="cal__chip cal__chip--card"
                    onClick={(ev) => {
                      ev.stopPropagation();
                      onOpenCard(c.cardId);
                    }}
                    title={`Задача: ${c.title}`}
                  >
                    📌 {c.title}
                  </span>
                ))}
              </span>
            </button>
          );
        })}
      </div>

      {creating && (
        <EventModal
          title="Новое событие"
          initial={{ startsAt: `${creating}T12:00`, color: "blue", title: "" }}
          busy={createEvent.isPending}
          onClose={() => setCreating(null)}
          onSubmit={(body) =>
            createEvent.mutate(body, {
              onSuccess: () => {
                toast.success("Событие создано");
                setCreating(null);
              },
              onError: () => toast.error("Не удалось создать событие"),
            })
          }
        />
      )}

      {editing && (
        <EventModal
          title="Событие"
          initial={{
            startsAt: toLocalInput(new Date(editing.startsAt)),
            endsAt: editing.endsAt ? toLocalInput(new Date(editing.endsAt)) : undefined,
            color: editing.color,
            title: editing.title,
          }}
          canEdit={editing.createdBy === me.id || me.roleLevel === 1 || me.id === group.createdBy}
          busy={updateEvent.isPending || deleteEvent.isPending}
          onClose={() => setEditing(null)}
          onSubmit={(body) =>
            updateEvent.mutate(
              { eventId: editing.id, body },
              {
                onSuccess: () => {
                  toast.success("Событие обновлено");
                  setEditing(null);
                },
                onError: () => toast.error("Не удалось обновить"),
              },
            )
          }
          onDelete={() =>
            deleteEvent.mutate(editing.id, {
              onSuccess: () => {
                toast.success("Событие удалено");
                setEditing(null);
              },
              onError: () => toast.error("Не удалось удалить"),
            })
          }
        />
      )}
    </div>
  );
}

interface ModalProps {
  title: string;
  initial: { title: string; startsAt: string; endsAt?: string; color: CalendarColor };
  canEdit?: boolean;
  busy: boolean;
  onClose: () => void;
  onSubmit: (body: { title: string; startsAt: string; endsAt?: string | null; color: CalendarColor }) => void;
  onDelete?: () => void;
}

function EventModal({ title, initial, canEdit = true, busy, onClose, onSubmit, onDelete }: ModalProps) {
  const [name, setName] = useState(initial.title);
  const [startsAt, setStartsAt] = useState(initial.startsAt);
  const [endsAt, setEndsAt] = useState(initial.endsAt ?? "");
  const [color, setColor] = useState<CalendarColor>(initial.color);
  const readOnly = !canEdit;

  const submit = () => {
    if (!name.trim()) {
      toast.error("Введите название");
      return;
    }
    if (endsAt && new Date(endsAt) < new Date(startsAt)) {
      toast.error("Конец не может быть раньше начала");
      return;
    }
    onSubmit({
      title: name.trim(),
      startsAt: new Date(startsAt).toISOString(),
      endsAt: endsAt ? new Date(endsAt).toISOString() : null,
      color,
    });
  };

  return (
    <Modal open title={title} onClose={onClose}>
      <Input label="Название" value={name} disabled={readOnly} onChange={(e) => setName(e.target.value)} autoFocus />
      <div className="ui-field">
        <label className="ui-field__label">Начало</label>
        <input className="ui-input" type="datetime-local" value={startsAt} disabled={readOnly} onChange={(e) => setStartsAt(e.target.value)} />
      </div>
      <div className="ui-field">
        <label className="ui-field__label">Конец (необязательно)</label>
        <input className="ui-input" type="datetime-local" value={endsAt} disabled={readOnly} onChange={(e) => setEndsAt(e.target.value)} />
      </div>
      <div className="ui-field">
        <label className="ui-field__label">Цвет</label>
        <div className="cal__palette">
          {CALENDAR_COLORS.map((c) => (
            <button
              key={c}
              type="button"
              disabled={readOnly}
              className={`cal__swatch${color === c ? " cal__swatch--on" : ""}`}
              style={{ background: COLOR_HEX[c] }}
              onClick={() => setColor(c)}
              aria-label={c}
            />
          ))}
        </div>
      </div>
      {canEdit && (
        <Button block loading={busy} onClick={submit}>
          Сохранить
        </Button>
      )}
      {onDelete && canEdit && (
        <Button variant="danger" block loading={busy} onClick={onDelete}>
          Удалить событие
        </Button>
      )}
      {readOnly && (
        <div style={{ fontSize: 13, color: "var(--color-text-tertiary)", textAlign: "center" }}>
          Только автор, владелец группы или CEO может изменять это событие.
        </div>
      )}
    </Modal>
  );
}
