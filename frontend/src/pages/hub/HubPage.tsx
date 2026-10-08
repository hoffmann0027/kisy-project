import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Icon } from "@shared/ui/icons";
import { AlertBadge } from "@shared/ui";
import { useNotifications } from "@entities/notification/queries";
import { useCapabilities } from "@shared/lib/useCapabilities";
import { feedLivesInHub } from "@shared/lib/nav";
import { NotificationsModal } from "@features/notifications/NotificationsModal";
import { NotesModal } from "@features/notes/NotesModal";
import { VotingModal } from "@features/voting/VotingModal";
import { FeedbackModal } from "@features/feedback/FeedbackModal";
import { NewGroupModal } from "@features/new-chat/NewGroupModal";
import { ConditionsModal } from "@features/conditions/ConditionsModal";
import { CallHistoryModal } from "@features/call/CallHistoryModal";
import { t } from "@shared/i18n";
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
      title: t("hub.page.notifications"),
      hint: unread > 0 ? t("hub.page.unread", { count: unread }) : t("hub.page.allRead"),
      icon: Icon.Bell,
      tint: "violet",
      // The one card that can be waiting on the user: a red count on its icon.
      alert: unread,
    },
    // The company vote board: the CEO runs it and everyone in the organisation
    // votes. An account that nobody invited is not part of that body — and the
    // server refuses /polls for it, so the card would only lead to an error.
    ...(caps.canVoteLevels
      ? [{ key: "voting" as const, title: t("hub.page.voting"), hint: t("hub.page.votingHint"), icon: Icon.Vote, tint: "orange" }]
      : []),
    // The feed's one door. An account without a rating board reaches it from
    // the tab bar and the rail instead, and then it must not also sit here —
    // one destination, one door (docs/spec/02-frontend-ux.md, audit D-11).
    ...(feedLivesInHub(caps)
      ? [
          {
            key: "feed" as const,
            title: t("hub.page.feed"),
            hint: t("hub.page.feedHint"),
            icon: Icon.Board,
            tint: "green",
            run: () => navigate("/feed"),
          },
        ]
      : []),
    { key: "notes" as const, title: t("hub.page.notes"), hint: t("hub.page.notesHint"), icon: Icon.Note, tint: "amber" },
    { key: "feedback" as const, title: t("hub.page.feedback"), hint: t("hub.page.feedbackHint"), icon: Icon.Feedback, tint: "green" },
    // Used to be reachable only from the desktop rail, i.e. not at all on a
    // phone.
    { key: "calls" as const, title: t("hub.page.calls"), hint: t("hub.page.callsHint"), icon: Icon.Phone, tint: "violet" },
  ];

  const actions = [
    { key: "group", label: t("hub.page.newGroup"), icon: Icon.FolderPlus, run: () => setModal("group") },
    ...(caps.canVoteLevels
      ? [{ key: "poll", label: t("hub.page.newPoll"), icon: Icon.Vote, run: () => setModal("voting") }]
      : []),
    { key: "note", label: t("hub.page.newNote"), icon: Icon.Edit, run: () => setModal("notes") },
    // Promotion is movement inside the role hierarchy, so it only exists for
    // accounts that are in it.
    ...(caps.canSeeConditions
      ? [{ key: "levels", label: t("hub.page.conditions"), icon: Icon.Levels, run: () => setModal("conditions") }]
      : []),
    // Invites and user management: the phone has no side rail, so the Hub and
    // the drawer are the two ways in. CEO only, as on the desktop.
    ...(caps.canAdmin
      ? [{ key: "admin", label: t("hub.page.admin"), icon: Icon.Shield, run: () => navigate("/admin") }]
      : []),
  ];

  return (
    <div className="hub">
      <div className="hub__scroll">
        <h1 className="hub__title">{t("hub.page.title")}</h1>

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
                {"alert" in c && c.alert ? (
                  <AlertBadge count={c.alert} label={t("hub.page.unread", { count: c.alert })} />
                ) : null}
              </span>
              <span className="hub-card__body">
                <span className="hub-card__title">{c.title}</span>
                <span className="hub-card__hint">{c.hint}</span>
              </span>
            </button>
          ))}
        </div>

        <div className="hub__section">{t("hub.page.quickActions")}</div>
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
