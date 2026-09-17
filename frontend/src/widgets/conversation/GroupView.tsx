import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { Icon } from "@shared/ui/icons";
import { roleLabel, type Group } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useGroupViewer } from "@entities/group/queries";
import { useBackHandler } from "@shared/lib/backStack";
import { Conversation } from "./Conversation";
import { BoardView } from "@widgets/board/BoardView";
import { CalendarView } from "@widgets/calendar/CalendarView";
import { GroupMembersModal } from "@features/profile/GroupMembersModal";
import { CommunityWall } from "@widgets/feed/CommunityWall";
import { VerifiedName } from "@shared/ui/VerifiedBadge";
import { useCapabilities } from "@shared/lib/useCapabilities";
import { SanctionsBanner } from "./SanctionsBanner";

type Tab = "chat" | "board" | "calendar" | "posts";

export function GroupView({ group }: { group: Group }) {
  const navigate = useNavigate();
  // A community opens on its wall; a group opens on its conversation.
  const isCommunity = group.kind === "community";
  const [tab, setTab] = useState<Tab>(isCommunity ? "posts" : "chat");
  const [membersOpen, setMembersOpen] = useState(false);

  // The board and the calendar are sub-views, not screens: back returns to the
  // group's chat rather than leaving the group.
  const homeTab: Tab = isCommunity ? "posts" : "chat";
  const isFounder = useAuthStore((s) => s.user?.id === group.createdBy);
  const { data: viewer } = useGroupViewer(group.id);
  // In a community the board and calendar are its editors' back office; the
  // server refuses them to plain readers, so the tabs are not offered either.
  // Until the viewer loads, a group shows its tabs as it always has, and a
  // community keeps them back rather than flash them at a reader.
  const canUseWorkspace = viewer?.canUseWorkspace ?? !isCommunity;
  // Losing the right while a sub-view is open (demoted, or the viewer loads
  // late) falls back to the home tab rather than to a screen of errors.
  const shown: Tab = (tab === "board" || tab === "calendar") && !canUseWorkspace ? homeTab : tab;
  useBackHandler(shown !== homeTab, () => setTab(homeTab));
  // People are added directly only to a group. A community is joined by the
  // reader's own choice — "Вступить" or a request its editors approve.
  const canAdd = isFounder && !isCommunity;
  // Who runs this group — and so is shown the CEO's sanctions on it.
  const caps = useCapabilities();
  const runsGroup =
    isFounder || caps.canAdmin || viewer?.role === "owner" || viewer?.role === "editor" || viewer?.role === "moderator";
  const banner = <SanctionsBanner groupId={group.id} runsGroup={runsGroup} />;
  // Editors-only group where the viewer is a plain member → composer hidden.
  const readOnly = viewer && !viewer.canPost ? "Писать могут только редакторы" : undefined;
  // Group's clearance, shown in the header so the level is visible in-chat.
  // "No threshold" is not a level, so it is not announced as one.
  const levelLabel = group.minRoleLevel === null
    ? (isCommunity ? "Сообщество" : "Группа")
    : `${isCommunity ? "Сообщество" : "Группа"} · от ${roleLabel(group.minRoleLevel)} и выше`;

  // On a phone this row wraps onto its own line under the header (see
  // .group-tabs in messenger.css), so the group name keeps the first line.
  const tabs = (
    <div className="group-tabs">
      {isCommunity && (
        <button
          className={cn("group-tab", shown === "posts" && "group-tab--active")}
          onClick={() => setTab("posts")}
        >
          Посты
        </button>
      )}
      {/* A community starts as a wall: the conversation tab appears only once
          its owners open posting to everyone (post_policy), which is the same
          question asked once instead of a second flag beside it. */}
      {(!isCommunity || group.postPolicy === "all") && (
        <button
          className={cn("group-tab", shown === "chat" && "group-tab--active")}
          onClick={() => setTab("chat")}
        >
          {isCommunity ? "Обсуждение" : "Чат"}
        </button>
      )}
      {canUseWorkspace && (
        <>
          <button
            className={cn("group-tab", shown === "board" && "group-tab--active")}
            onClick={() => setTab("board")}
          >
            <Icon.Board size={16} /> Доска
          </button>
          <button
            className={cn("group-tab", shown === "calendar" && "group-tab--active")}
            onClick={() => setTab("calendar")}
          >
            <Icon.Calendar size={16} /> Календарь
          </button>
        </>
      )}
      <button className="group-tab" onClick={() => setMembersOpen(true)} title="Участники">
        <Icon.Users size={16} />
      </button>
    </div>
  );

  const members = (
    <GroupMembersModal group={group} canAdd={canAdd} open={membersOpen} onClose={() => setMembersOpen(false)} />
  );

  if (shown === "posts") {
    return (
      <section className="conv">
        <header className="conv__header">
          <button className="conv__back" title="Назад" onClick={() => navigate("/communities")}>
            <Icon.Back size={22} />
          </button>
          <div className="conv__header-body">
            <div className="conv__title">
              <VerifiedName name={group.name} verified={!!group.verifiedAt} subject="group" />
            </div>
            <div className="conv__status">{group.isPublic ? "Открытое сообщество" : "Закрытое сообщество"}</div>
          </div>
          {tabs}
        </header>
        {banner}
        <CommunityWall
          group={group}
          canPost={viewer?.canPost ?? false}
          membersOnly={!group.isPublic && !caps.canAdmin && viewer !== undefined && !viewer.member}
        />
        {members}
      </section>
    );
  }

  if (shown === "board" || shown === "calendar") {
    return (
      <section className="conv">
        <header className="conv__header">
          <button className="conv__back" title="Назад" onClick={() => navigate("/communities")}>
            <Icon.Back size={22} />
          </button>
          <div className="conv__header-body">
            <div className="conv__title">
              <VerifiedName name={group.name} verified={!!group.verifiedAt} subject="group" />
            </div>
            <div className="conv__status">{shown === "board" ? "Доска задач" : "Календарь"}</div>
          </div>
          {tabs}
        </header>
        {banner}
        {shown === "board" ? <BoardView group={group} /> : <CalendarView group={group} onOpenCard={() => setTab("board")} />}
        {members}
      </section>
    );
  }

  return (
    <>
      <Conversation
        target={{
          chatType: "group",
          chatId: group.id,
          title: group.name,
          verified: !!group.verifiedAt,
          verifiedSubject: "group",
          avatarName: group.name,
          avatarUrl: group.avatarUrl,
          offlineLabel: levelLabel,
        }}
        headerActions={tabs}
        readOnly={readOnly}
        banner={banner}
      />
      {members}
    </>
  );
}
