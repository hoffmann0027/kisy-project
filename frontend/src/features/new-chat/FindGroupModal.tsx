import { useNavigate } from "react-router-dom";
import { Avatar, Button, Modal, Spinner, toast } from "@shared/ui";
import { roleLabel, type DirectoryGroup } from "@shared/api/types";
import { useGroupDirectory, useJoinGroup } from "@entities/group/queries";

interface Props {
  open: boolean;
  onClose: () => void;
}

// FindGroupModal is the "find a group" catalogue: every group the user is
// cleared to see and is not already a member of. Open groups join instantly;
// request groups show "Подать заявку" / "Заявка на рассмотрении".
export function FindGroupModal({ open, onClose }: Props) {
  const { data: groups, isPending } = useGroupDirectory(open);
  const join = useJoinGroup();
  const navigate = useNavigate();

  const act = (g: DirectoryGroup) => {
    join.mutate(g.id, {
      onSuccess: ({ joined }) => {
        if (joined) {
          toast.success("Вы вступили в группу");
          onClose();
          navigate(`/group/${g.id}`);
        } else {
          toast.success("Заявка отправлена");
        }
      },
      onError: () => toast.error("Не удалось вступить"),
    });
  };

  return (
    <Modal open={open} title="Найти группу" onClose={onClose}>
      {isPending && (
        <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
          <Spinner />
        </div>
      )}
      {!isPending && (groups?.length ?? 0) === 0 && (
        <div style={{ color: "var(--color-text-secondary)", fontSize: 14, padding: "12px 0" }}>
          Нет доступных для вступления групп.
        </div>
      )}
      <div style={{ maxHeight: 380, overflowY: "auto", display: "flex", flexDirection: "column", gap: 4 }}>
        {groups?.map((g) => {
          const pending = g.requestStatus === "pending";
          return (
            <div key={g.id} className="user-row" style={{ cursor: "default" }}>
              <Avatar name={g.name} url={g.avatarUrl} size={40} />
              <div style={{ flex: 1 }}>
                <div className="user-row__name">{g.name}</div>
                <div className="user-row__role">
                  {g.joinPolicy === "open" ? "Публичная" : "Закрытая (по заявке)"} · от {roleLabel(g.minRoleLevel)} и выше
                </div>
              </div>
              {pending ? (
                <span style={{ fontSize: 13, color: "var(--color-text-tertiary)" }}>Заявка на рассмотрении</span>
              ) : (
                <Button variant="secondary" loading={join.isPending} onClick={() => act(g)}>
                  {g.joinPolicy === "open" ? "Вступить" : "Подать заявку"}
                </Button>
              )}
            </div>
          );
        })}
      </div>
    </Modal>
  );
}
