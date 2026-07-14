import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { Icon } from "@shared/ui/icons";
import { roleLabel, type Group } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useGroupViewer } from "@entities/group/queries";
import { Conversation } from "./Conversation";
import { BoardView } from "@widgets/board/BoardView";
import { GroupMembersModal } from "@features/profile/GroupMembersModal";

type Tab = "chat" | "board";

export function GroupView({ group }: { group: Group }) {
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>("chat");
  const [membersOpen, setMembersOpen] = useState(false);
  const isFounder = useAuthStore((s) => s.user?.id === group.createdBy);
  const { data: viewer } = useGroupViewer(group.id);
  // Editors-only group where the viewer is a plain member → composer hidden.
  const readOnly = viewer && !viewer.canPost ? "Писать могут только редакторы" : undefined;
  // Group's clearance, shown in the header so the level is visible in-chat.
  const levelLabel = `Группа · от ${roleLabel(group.minRoleLevel)} и выше`;

  const tabs = (
    <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
      <button
        className={cn("group-tab", tab === "chat" && "group-tab--active")}
        onClick={() => setTab("chat")}
      >
        Чат
      </button>
      <button
        className={cn("group-tab", tab === "board" && "group-tab--active")}
        onClick={() => setTab("board")}
      >
        <Icon.Board size={16} /> Доска
      </button>
      <button className="group-tab" onClick={() => setMembersOpen(true)} title="Участники">
        <Icon.Users size={16} />
      </button>
    </div>
  );

  if (tab === "board") {
    return (
      <section className="conv">
        <header className="conv__header">
          <button className="conv__back" title="Назад" onClick={() => navigate("/")}>
            <Icon.Back size={22} />
          </button>
          <div className="conv__header-body">
            <div className="conv__title">{group.name}</div>
            <div className="conv__status">Доска задач</div>
          </div>
          {tabs}
        </header>
        <BoardView group={group} />
        <GroupMembersModal group={group} canAdd={isFounder} open={membersOpen} onClose={() => setMembersOpen(false)} />
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
          avatarName: group.name,
          avatarUrl: group.avatarUrl,
          offlineLabel: levelLabel,
        }}
        headerActions={tabs}
        readOnly={readOnly}
      />
      <GroupMembersModal group={group} canAdd={isFounder} open={membersOpen} onClose={() => setMembersOpen(false)} />
    </>
  );
}
