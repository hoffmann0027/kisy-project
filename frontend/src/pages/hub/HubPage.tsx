import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Icon } from "@shared/ui/icons";
import { useNotifications } from "@entities/notification/queries";
import { NotificationsModal } from "@features/notifications/NotificationsModal";
import { NotesModal } from "@features/notes/NotesModal";
import { VotingModal } from "@features/voting/VotingModal";
import { FeedbackModal } from "@features/feedback/FeedbackModal";
import { NewGroupModal } from "@features/new-chat/NewGroupModal";
import { ConditionsModal } from "@features/conditions/ConditionsModal";
import "./hub.css";

// The Hub (design_handoff_kisy_mobile §5): the phone layout has room for four
// tabs, so everything secondary lives here — the screens that used to hang off
// the desktop rail. Each card opens the feature's existing modal; nothing is
// reimplemented.

type Modal = "notifications" | "voting" | "notes" | "feedback" | "conditions" | "group" | null;

export function HubPage() {
  const navigate = useNavigate();
  const [modal, setModal] = useState<Modal>(null);
  const { data: notif } = useNotifications();
  const unread = notif?.unreadCount ?? 0;

  const cards = [
    {
      key: "notifications" as const,
      title: "Уведомления",
      hint: unread > 0 ? `${unread} ${plural(unread, "новое", "новых", "новых")}` : "Всё прочитано",
      icon: Icon.Bell,
      tint: "violet",
    },
    { key: "voting" as const, title: "Голосования", hint: "Опросы команды", icon: Icon.Vote, tint: "orange" },
    { key: "notes" as const, title: "Заметки", hint: "Личные записи", icon: Icon.Note, tint: "amber" },
    { key: "feedback" as const, title: "Отзывы", hint: "Идеи и проблемы", icon: Icon.Feedback, tint: "green" },
  ];

  const actions = [
    { key: "group", label: "Новая группа", icon: Icon.FolderPlus, run: () => setModal("group") },
    { key: "poll", label: "Создать опрос", icon: Icon.Vote, run: () => setModal("voting") },
    { key: "note", label: "Новая заметка", icon: Icon.Edit, run: () => setModal("notes") },
    { key: "levels", label: "Условия повышения", icon: Icon.Levels, run: () => setModal("conditions") },
  ];

  return (
    <div className="hub">
      <div className="hub__scroll">
        <h1 className="hub__title">Хаб</h1>

        <div className="hub__grid">
          {cards.map((c) => (
            <button key={c.key} type="button" className="hub-card" onClick={() => setModal(c.key)}>
              <span className={`hub-card__badge hub-card__badge--${c.tint}`}>
                <c.icon size={22} />
              </span>
              <span className="hub-card__body">
                <span className="hub-card__title">{c.title}</span>
                <span className="hub-card__hint">{c.hint}</span>
              </span>
            </button>
          ))}
        </div>

        <div className="hub__section">Быстрые действия</div>
        <div className="hub__actions">
          {actions.map((a) => (
            <button key={a.key} type="button" className="hub-action" onClick={a.run}>
              <span className="hub-action__icon">
                <a.icon size={22} />
              </span>
              <span className="hub-action__label">{a.label}</span>
              <span className="hub-action__chevron">
                <Icon.Chevron size={18} />
              </span>
            </button>
          ))}
        </div>
      </div>

      <NotificationsModal open={modal === "notifications"} onClose={() => setModal(null)} />
      <VotingModal open={modal === "voting"} onClose={() => setModal(null)} />
      <NotesModal open={modal === "notes"} onClose={() => setModal(null)} />
      <FeedbackModal open={modal === "feedback"} onClose={() => setModal(null)} />
      <ConditionsModal open={modal === "conditions"} onClose={() => setModal(null)} />
      <NewGroupModal
        open={modal === "group"}
        onClose={() => setModal(null)}
        onCreated={(g) => navigate(`/group/${g.id}`)}
      />
    </div>
  );
}

/** Russian plural for the unread counter (1 новое, 2 новых, 5 новых). */
function plural(n: number, one: string, few: string, many: string): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return one;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few;
  return many;
}
