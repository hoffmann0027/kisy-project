import { useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { ROLE_LABELS, type Group, type GroupKind } from "@shared/api/types";
import { useCreateGroup } from "@entities/group/queries";
import { useAuthStore } from "@shared/store/auth";
import { useCapabilities } from "@shared/lib/useCapabilities";

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: (group: Group) => void;
}

export function NewGroupModal({ open, onClose, onCreated }: Props) {
  const myLevel = useAuthStore((s) => s.user?.roleLevel ?? null);
  const caps = useCapabilities();
  const [kind, setKind] = useState<GroupKind>("group");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isPublic, setIsPublic] = useState(true);
  // Only ever consulted for an account that has a level (see below).
  const [minRoleLevel, setMinRoleLevel] = useState(myLevel ?? 10);
  const create = useCreateGroup();

  // A user may only create a group whose minimum clearance is their own
  // level or weaker (numerically >= their level).
  const levelOptions = Object.entries(ROLE_LABELS).filter(([lvl]) => Number(lvl) >= (myLevel ?? 1));

  const submit = () => {
    if (name.trim().length < 1) {
      toast.error(kind === "community" ? "Введите название сообщества" : "Введите название группы");
      return;
    }
    create.mutate(
      {
        name: name.trim(),
        // An account outside the hierarchy has no threshold to choose, so it
        // sends none: the group is open to everyone.
        minRoleLevel: caps.canSeeLevels ? minRoleLevel : null,
        description: description.trim() || undefined,
        kind,
        isPublic: kind === "community" && isPublic,
      },
      {
        onSuccess: ({ group }) => {
          toast.success(kind === "community" ? "Сообщество создано" : "Группа создана");
          setName("");
          setDescription("");
          onCreated(group);
          onClose();
        },
        onError: () => toast.error("Не удалось создать"),
      },
    );
  };

  return (
    <Modal open={open} title={kind === "community" ? "Новое сообщество" : "Новая группа"} onClose={onClose}>
      <div className="ui-field">
        <label className="ui-field__label">Что создаём</label>
        <div style={{ display: "flex", gap: 8 }}>
          <Button variant={kind === "group" ? "primary" : "secondary"} onClick={() => setKind("group")}>
            Группа
          </Button>
          <Button variant={kind === "community" ? "primary" : "secondary"} onClick={() => setKind("community")}>
            Сообщество
          </Button>
        </div>
        <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
          {kind === "group"
            ? "Группа — общий чат: писать могут все участники."
            : "Сообщество — стена с постами: публикуют владельцы и редакторы, остальные читают и ставят реакции. Обсуждение можно включить позже в настройках."}
        </span>
      </div>

      <Input
        label="Название"
        placeholder={kind === "community" ? "Например, Новости компании" : "Например, Отдел разработки"}
        value={name}
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />
      <Input label="Описание (необязательно)" value={description} onChange={(e) => setDescription(e.target.value)} />

      {/* No level, no selector. An empty or disabled dropdown would be asking a
          question this account cannot answer; its groups are simply open. */}
      {caps.canSeeLevels && (
        <div className="ui-field">
          <label className="ui-field__label">Минимальный уровень доступа</label>
          <select className="ui-input" value={minRoleLevel} onChange={(e) => setMinRoleLevel(Number(e.target.value))}>
            {levelOptions.map(([lvl, label]) => (
              <option key={lvl} value={lvl}>
                {lvl}. {label}
              </option>
            ))}
          </select>
          <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
            Будет видно пользователям этого уровня и выше. Нельзя создать с доступом выше вашего уровня.
          </span>
        </div>
      )}

      {kind === "community" && (
        <label className="ui-field" style={{ flexDirection: "row", alignItems: "center", gap: 10 }}>
          <input type="checkbox" checked={isPublic} onChange={(e) => setIsPublic(e.target.checked)} />
          <span>
            <span style={{ display: "block" }}>Открытое сообщество</span>
            <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
              Посты попадают в общую ленту, и вступить может любой.
            </span>
          </span>
        </label>
      )}

      <Button block loading={create.isPending} onClick={submit}>
        {kind === "community" ? "Создать сообщество" : "Создать группу"}
      </Button>
    </Modal>
  );
}
