import { useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { ROLE_LABELS, type Group } from "@shared/api/types";
import { useCreateGroup } from "@entities/group/queries";
import { useAuthStore } from "@shared/store/auth";

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: (group: Group) => void;
}

export function NewGroupModal({ open, onClose, onCreated }: Props) {
  const myLevel = useAuthStore((s) => s.user?.roleLevel ?? 10);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [minRoleLevel, setMinRoleLevel] = useState(myLevel);
  const create = useCreateGroup();

  // A user may only create a group whose minimum clearance is their own
  // level or weaker (numerically >= their level).
  const levelOptions = Object.entries(ROLE_LABELS).filter(([lvl]) => Number(lvl) >= myLevel);

  const submit = () => {
    if (name.trim().length < 1) {
      toast.error("Введите название группы");
      return;
    }
    create.mutate(
      { name: name.trim(), minRoleLevel, description: description.trim() || undefined },
      {
        onSuccess: ({ group }) => {
          toast.success("Группа создана");
          setName("");
          setDescription("");
          onCreated(group);
          onClose();
        },
        onError: () => toast.error("Не удалось создать группу"),
      },
    );
  };

  return (
    <Modal open={open} title="Новая группа" onClose={onClose}>
      <Input label="Название" placeholder="Например, Отдел разработки" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      <Input label="Описание (необязательно)" value={description} onChange={(e) => setDescription(e.target.value)} />
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
          Группа будет видна пользователям этого уровня и выше. Нельзя создать группу с доступом выше вашего уровня.
        </span>
      </div>
      <Button block loading={create.isPending} onClick={submit}>
        Создать группу
      </Button>
    </Modal>
  );
}
