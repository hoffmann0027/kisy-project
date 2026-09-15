import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Icon } from "@shared/ui/icons";
import { useNotifications } from "@entities/notification/queries";
import { useCapabilities } from "@shared/lib/useCapabilities";
import { NotificationsModal } from "@features/notifications/NotificationsModal";
import { NotesModal } from "@features/notes/NotesModal";
import { VotingModal } from "@features/voting/VotingModal";
import { FeedbackModal } from "@features/feedback/FeedbackModal";
import { NewGroupModal } from "@features/new-chat/NewGroupModal";
import { ConditionsModal } from "@features/conditions/ConditionsModal";
import { CallHistoryModal } from "@features/call/CallHistoryModal";
import "./hub.css";

// The Hub (design_handoff_kisy_mobile §5): the phone layout has room for four
// tabs, so everything secondary lives here — the screens that used to hang off
// the desktop rail. Each card opens the feature's existing modal; nothing is
// reimplemented.

type Modal = "notifications" | "voting" | "notes" | "feedback" | "conditions" | "group" | "calls" | null;

export function HubPage() {
  const navigate = useNavigate();
  const [modal, setModal] = useState<Modal>(null);
  const { data: notif } = useNotifications();
  const caps = useCapabilities();
  const unread = notif?.unreadCount ?? 0;

  const cards = [
    {
      key: "notifications" as const,
      title: "Уведомления",
      hint: unread > 0 ? `${unread} ${plural(unread, "новое", "новых", "новых")}` : "Всё прочитано",
      icon: Icon.Bell,
      tint: "violet",
    },
    // The company vote board: the CEO runs it and everyone in the organisation
    // votes. An account that nobody invited is not part of that body — and the
    // server refuses /polls for it, so the card would only lead to an error.
    ...(caps.canVoteLevels
      ? [{ key: "voting" as const, title: "Голосования", hint: "Опросы команды", icon: Icon.Vote, tint: "orange" }]
      : []),
    // The feed's one door. An account without a rating board reaches it from
    // the tab bar instead, and then it must not also sit here — one
    // destination, one door (docs/spec/02-frontend-ux.md).
    ...(caps.feedPlacement === "hub"
      ? [
          {
            key: "feed" as const,
            title: "Лента",
            hint: "Посты сообществ",
            icon: Icon.Board,
            tint: "green",
            run: () => navigate("/feed"),
          },
        ]
      : []),
    { key: "notes" as const, title: "Заметки", hint: "Личные записи", icon: Icon.Note, tint: "amber" },
    { key: "feedback" as const, title: "Отзывы", hint: "Идеи и проблемы", icon: Icon.Feedback, tint: "green" },
    // Used to be reachable only from the desktop rail, i.e. not at all on a
    // phone.
    { key: "calls" as const, title: "Звонки", hint: "История вызовов", icon: Icon.Phone, tint: "violet" },
  ];

  const actions = [
    // Creating a group means choosing the clearance it requires, and an
    // account outside the hierarchy has none to choose from — the server
    // refuses it. Groups without a threshold, which is what such an account
    // will create, arrive with communities in the next step; until then the
    // action is hidden rather than left to fail.
    ...(caps.isInvited
      ? [{ key: "group", label: "Новая группа", icon: Icon.FolderPlus, run: () => setModal("group") }]
      : []),
    ...(caps.canVoteLevels
      ? [{ key: "poll", label: "Создать опрос", icon: Icon.Vote, run: () => setModal("voting") }]
      : []),
    { key: "note", label: "Новая заметка", icon: Icon.Edit, run: () => setModal("notes") },
    // Promotion is movement inside the role hierarchy, so it only exists for
    // accounts that are in it.
    ...(caps.canSeeConditions
      ? [{ key: "levels", label: "Условия повышения", icon: Icon.Levels, run: () => setModal("conditions") }]
      : []),
    // Invites and user management: the phone has no side rail, so the Hub and
    // the drawer are the two ways in. CEO only, as on the desktop.
    ...(caps.canAdmin
      ? [{ key: "admin", label: "Администрирование", icon: Icon.Shield, run: () => navigate("/admin") }]
      : []),
  ];

  return (
    <div className="hub">
      <div className="hub__scroll">
        <h1 className="hub__title">Хаб</h1>

        <div className="hub__grid">
          {cards.map((c) => (
            <button
              key={c.key}
              type="button"
              className="hub-card"
              // Most cards open a dialog; a card that leads to a screen of its
              // own says so with its own `run`.
              onClick={() => ("run" in c && c.run ? c.run() : setModal(c.key as Modal))}
            >
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
      <CallHistoryModal open={modal === "calls"} onClose={() => setModal(null)} />
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
