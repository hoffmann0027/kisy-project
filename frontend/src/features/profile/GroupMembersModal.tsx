import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Avatar, Button, Modal, Spinner, toast } from "@shared/ui";
import { ROLE_LABELS, roleLabel, type Group, type GroupRole, type JoinPolicy, type PostPolicy } from "@shared/api/types";
import { groupsApi, usersApi } from "@shared/api/endpoints";
import {
  groupKeys,
  useAddMember,
  useDecideRequest,
  useDeleteGroup,
  useGroupMembers,
  useGroupRequests,
  useSetMemberRole,
  useUpdateGroupLevel,
  useUpdateGroupSettings,
} from "@entities/group/queries";
import { useAuthStore } from "@shared/store/auth";
import { ApiError } from "@shared/api/envelope";
import { AvatarCropper } from "./AvatarCropper";

interface Props {
  group: Group;
  canAdd: boolean;
  open: boolean;
  onClose: () => void;
}

const EDITOR_TIER: GroupRole[] = ["owner", "editor", "moderator"];

// Human labels for the in-group roles.
const GROUP_ROLE_LABEL: Record<GroupRole, string> = {
  owner: "владелец",
  editor: "редактор",
  moderator: "модератор",
  member: "участник",
};

export function GroupMembersModal({ group, canAdd, open, onClose }: Props) {
  const { data: members, isPending } = useGroupMembers(open ? group.id : null);
  const [adding, setAdding] = useState(false);
  const navigate = useNavigate();
  const del = useDeleteGroup();
  const qc = useQueryClient();
  const me = useAuthStore((s) => s.user!);
  const updateLevel = useUpdateGroupLevel();
  const updateSettings = useUpdateGroupSettings();
  const setRole = useSetMemberRole();
  // The CEO may manage any group; the founder their own.
  const isCEO = me.roleLevel === 1;
  const isOwner = isCEO || me.id === group.createdBy;
  const canManage = isOwner;
  const canDelete = canManage;
  // My own in-group role (for the editor tier → may approve requests).
  const myRole = members?.find((m) => m.user.id === me.id)?.role;
  const canApprove = isOwner || (myRole !== undefined && EDITOR_TIER.includes(myRole));

  const { data: requests } = useGroupRequests(group.id, open && canApprove);
  const decide = useDecideRequest();

  const changeLevel = (level: number) => {
    if (level === group.minRoleLevel) return;
    updateLevel.mutate(
      { groupId: group.id, minRoleLevel: level },
      { onSuccess: () => toast.success("Уровень группы изменён"), onError: () => toast.error("Не удалось изменить уровень") },
    );
  };

  const changeAccess = (value: string) => {
    const [joinPolicy, postPolicy] = value.split(":") as [JoinPolicy, PostPolicy];
    if (joinPolicy === group.joinPolicy && postPolicy === group.postPolicy) return;
    updateSettings.mutate(
      { groupId: group.id, joinPolicy, postPolicy },
      { onSuccess: () => toast.success("Настройки доступа обновлены"), onError: () => toast.error("Не удалось обновить доступ") },
    );
  };

  const toggleEditor = (userId: string, current: GroupRole) => {
    const role: GroupRole = current === "editor" ? "member" : "editor";
    setRole.mutate(
      { groupId: group.id, userId, role },
      { onError: () => toast.error("Не удалось изменить роль") },
    );
  };

  const uploadGroupAvatar = async (blob: Blob) => {
    await groupsApi.uploadAvatar(group.id, blob);
    qc.invalidateQueries({ queryKey: groupKeys.list });
  };

  const removeGroup = () => {
    if (!window.confirm(`Удалить группу «${group.name}»? Это удалит её чат и доску задач.`)) return;
    del.mutate(group.id, {
      onSuccess: () => {
        toast.success("Группа удалена");
        onClose();
        navigate("/", { replace: true });
      },
      onError: () => toast.error("Не удалось удалить группу"),
    });
  };

  const pendingCount = requests?.length ?? 0;

  return (
    <Modal open={open} title={`Участники · ${group.name}`} onClose={onClose}>
      <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
        {canManage ? (
          <AvatarCropper name={group.name} url={group.avatarUrl} size={56} onUpload={uploadGroupAvatar} />
        ) : (
          <Avatar name={group.name} url={group.avatarUrl} size={56} />
        )}
        <div>
          <div style={{ fontWeight: 640, fontSize: 17 }}>{group.name}</div>
          <div style={{ color: "var(--color-text-secondary)", fontSize: 14 }}>
            {canManage ? "Нажмите на аватар, чтобы изменить" : "Группа"}
          </div>
        </div>
      </div>

      <div className="ui-field">
        <label className="ui-field__label">Уровень доступа</label>
        {isCEO ? (
          <select className="ui-input" value={group.minRoleLevel} disabled={updateLevel.isPending} onChange={(e) => changeLevel(Number(e.target.value))}>
            {Object.entries(ROLE_LABELS).map(([lvl, label]) => (
              <option key={lvl} value={lvl}>
                {lvl}. {label}
              </option>
            ))}
          </select>
        ) : (
          <div className="ui-input" style={{ display: "flex", alignItems: "center" }}>
            {group.minRoleLevel}. {roleLabel(group.minRoleLevel)}
          </div>
        )}
        <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
          {isCEO
            ? "Как CEO вы можете менять уровень группы. Она видна пользователям выбранного уровня и выше."
            : "Группа видна пользователям этого уровня и выше."}
        </span>
      </div>

      {canManage && (
        <div className="ui-field">
          <label className="ui-field__label">Доступ</label>
          <select
            className="ui-input"
            value={`${group.joinPolicy}:${group.postPolicy}`}
            disabled={updateSettings.isPending}
            onChange={(e) => changeAccess(e.target.value)}
          >
            <option value="open:all">Публичная · пишут все</option>
            <option value="open:editors">Публичная · пишут редакторы (канал)</option>
            <option value="request:all">Закрытая (по заявке) · пишут все</option>
            <option value="request:editors">Закрытая (по заявке) · пишут редакторы (канал)</option>
          </select>
          <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
            Публичная — вступают сразу; закрытая — по заявке. «Пишут редакторы» — остальные только читают. Клиренс всегда сильнее этих настроек.
          </span>
        </div>
      )}

      {canApprove && pendingCount > 0 && (
        <div className="ui-field">
          <label className="ui-field__label">Заявки на вступление · {pendingCount}</label>
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {requests?.map((u) => (
              <div key={u.id} className="user-row" style={{ cursor: "default" }}>
                <Avatar name={u.displayName} url={u.avatarUrl} size={34} />
                <div style={{ flex: 1 }}>
                  <div className="user-row__name">{u.displayName}</div>
                  <div className="user-row__role">@{u.username} · {roleLabel(u.roleLevel)}</div>
                </div>
                <div style={{ display: "flex", gap: 6 }}>
                  <Button variant="secondary" loading={decide.isPending} onClick={() => decide.mutate({ groupId: group.id, userId: u.id, approve: true })}>
                    Принять
                  </Button>
                  <Button variant="ghost" onClick={() => decide.mutate({ groupId: group.id, userId: u.id, approve: false })}>
                    Отклонить
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {canAdd && !adding && (
        <Button variant="secondary" onClick={() => setAdding(true)}>
          Добавить участника
        </Button>
      )}
      {adding && <AddMemberPicker group={group} onDone={() => setAdding(false)} />}

      <div style={{ maxHeight: 320, overflowY: "auto", display: "flex", flexDirection: "column", gap: 2 }}>
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
            <Spinner />
          </div>
        )}
        {members?.map((m) => {
          const founder = m.user.id === group.createdBy;
          return (
            <div key={m.user.id} className="user-row" style={{ cursor: "default" }}>
              <Avatar name={m.user.displayName} url={m.user.avatarUrl} size={38} />
              <div style={{ flex: 1 }}>
                <div className="user-row__name">{m.user.displayName}</div>
                <div className="user-row__role">
                  @{m.user.username} · {roleLabel(m.user.roleLevel)}
                  {founder ? " · основатель" : m.role !== "member" ? ` · ${GROUP_ROLE_LABEL[m.role]}` : ""}
                </div>
              </div>
              {canManage && !founder && (
                <Button variant="ghost" loading={setRole.isPending} onClick={() => toggleEditor(m.user.id, m.role)}>
                  {m.role === "editor" ? "Снять редактора" : "Сделать редактором"}
                </Button>
              )}
            </div>
          );
        })}
      </div>

      {canDelete && (
        <div style={{ borderTop: "1px solid var(--color-border)", paddingTop: 14 }}>
          <Button variant="danger" block loading={del.isPending} onClick={removeGroup}>
            Удалить группу
          </Button>
        </div>
      )}
    </Modal>
  );
}

function AddMemberPicker({ group, onDone }: { group: Group; onDone: () => void }) {
  const [query, setQuery] = useState("");
  const add = useAddMember();
  const { data } = useQuery({
    queryKey: ["directory", "group-add", query],
    queryFn: async () => (await usersApi.directory(query)).users,
  });

  const pick = (userId: string) => {
    add.mutate(
      { groupId: group.id, userId },
      {
        onSuccess: () => {
          toast.success("Участник добавлен");
          onDone();
        },
        onError: (e) =>
          toast.error(e instanceof ApiError && e.status === 409 ? "Уже в группе" : "Не удалось добавить (проверьте уровень доступа)"),
      },
    );
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      <input className="ui-input" placeholder="Поиск по имени" autoFocus value={query} onChange={(e) => setQuery(e.target.value)} />
      <div style={{ maxHeight: 200, overflowY: "auto" }}>
        {data?.map((u) => (
          <button key={u.id} className="user-row" onClick={() => pick(u.id)} disabled={add.isPending}>
            <Avatar name={u.displayName} url={u.avatarUrl} size={34} />
            <div>
              <div className="user-row__name">{u.displayName}</div>
              <div className="user-row__role">
                @{u.username} · {roleLabel(u.roleLevel)}
              </div>
            </div>
          </button>
        ))}
      </div>
      <Button variant="ghost" onClick={onDone}>
        Готово
      </Button>
    </div>
  );
}
