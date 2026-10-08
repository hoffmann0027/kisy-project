import { useNavigate } from "react-router-dom";
import { t } from "@shared/i18n";
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
          toast.success(t("chat.findGroup.joined"));
          onClose();
          navigate(`/group/${g.id}`);
        } else {
          toast.success(t("chat.findGroup.requestSent"));
        }
      },
      onError: () => toast.error(t("chat.findGroup.joinFailed")),
    });
  };

  return (
    <Modal open={open} title={t("chat.list.findGroup")} onClose={onClose}>
      {isPending && (
        <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
          <Spinner />
        </div>
      )}
      {!isPending && (groups?.length ?? 0) === 0 && (
        <div style={{ color: "var(--color-text-secondary)", fontSize: 14, padding: "12px 0" }}>
          {t("chat.findGroup.empty")}
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
                  {t(g.joinPolicy === "open" ? "chat.findGroup.openFromLevel" : "chat.findGroup.requestFromLevel", {
                    level: roleLabel(g.minRoleLevel),
                  })}
                </div>
              </div>
              {pending ? (
                <span style={{ fontSize: 13, color: "var(--color-text-tertiary)" }}>{t("chat.findGroup.requestPending")}</span>
              ) : (
                <Button variant="secondary" loading={join.isPending} onClick={() => act(g)}>
                  {g.joinPolicy === "open" ? t("chat.findGroup.join") : t("chat.findGroup.requestToJoin")}
                </Button>
              )}
            </div>
          );
        })}
      </div>
    </Modal>
  );
}
