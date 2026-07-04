import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Avatar, Button, Modal, Spinner, toast } from "@shared/ui";
import { roleLabel, type Group } from "@shared/api/types";
import { groupsApi, usersApi } from "@shared/api/endpoints";
import { groupKeys, useAddMember, useDeleteGroup, useGroupMembers } from "@entities/group/queries";
import { useAuthStore } from "@shared/store/auth";
import { ApiError } from "@shared/api/envelope";
import { AvatarCropper } from "./AvatarCropper";

interface Props {
  group: Group;
  canAdd: boolean;
  open: boolean;
  onClose: () => void;
}

export function GroupMembersModal({ group, canAdd, open, onClose }: Props) {
  const { data: members, isPending } = useGroupMembers(open ? group.id : null);
  const [adding, setAdding] = useState(false);
  const navigate = useNavigate();
  const del = useDeleteGroup();
  const qc = useQueryClient();
  const me = useAuthStore((s) => s.user!);
  // The CEO may delete/manage any group; the founder their own.
  const canManage = me.roleLevel === 1 || me.id === group.createdBy;
  const canDelete = canManage;

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
        {members?.map((u) => (
          <div key={u.id} className="user-row" style={{ cursor: "default" }}>
            <Avatar name={u.displayName} url={u.avatarUrl} size={38} />
            <div>
              <div className="user-row__name">{u.displayName}</div>
              <div className="user-row__role">
                @{u.username} · {roleLabel(u.roleLevel)}
                {u.id === group.createdBy ? " · основатель" : ""}
              </div>
            </div>
          </div>
        ))}
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
