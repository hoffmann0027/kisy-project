import { useState } from "react";
import { Button, Modal, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useIsBlocked, useSetBlocked } from "@entities/block/queries";

// Blocking from the conversation itself — where you are when you decide you
// have had enough of someone. The dialog spells out what changes, because a
// block is silent and one-sided: nothing visible happens on the other side,
// and people should not have to guess at that.

interface Props {
  userId: string;
  name: string;
}

export function BlockButton({ userId, name }: Props) {
  const blocked = useIsBlocked(userId);
  const setBlocked = useSetBlocked();
  const [open, setOpen] = useState(false);

  const apply = () => {
    setBlocked.mutate(
      { userId, blocked: !blocked },
      {
        onSuccess: () => {
          toast.success(blocked ? "Пользователь разблокирован" : "Пользователь заблокирован");
          setOpen(false);
        },
        onError: () => toast.error("Не удалось изменить блокировку"),
      },
    );
  };

  return (
    <>
      <button
        className="conv__panel-toggle"
        title={blocked ? `Разблокировать ${name}` : `Заблокировать ${name}`}
        onClick={() => setOpen(true)}
      >
        <Icon.Ban size={20} />
      </button>

      {open && (
        <Modal open title={blocked ? "Разблокировать" : "Заблокировать"} onClose={() => setOpen(false)}>
          {blocked ? (
            <p className="block-dialog__text">
              {name} снова сможет писать вам и звонить, и вы увидите их записи в ленте.
            </p>
          ) : (
            <>
              <p className="block-dialog__text">После блокировки {name}:</p>
              <ul className="block-dialog__list">
                <li>не сможет писать вам и звонить — и вы им тоже;</li>
                <li>исчезнет из вашей ленты и из поиска, а вы — из их;</li>
                <li>не узнает о блокировке: уведомления не будет.</li>
              </ul>
              <p className="block-dialog__text">Снять блокировку можно в профиле в любой момент.</p>
            </>
          )}
          <div className="block-dialog__actions">
            <Button variant="secondary" block onClick={() => setOpen(false)}>
              Отмена
            </Button>
            <Button
              variant={blocked ? "primary" : "danger"}
              block
              loading={setBlocked.isPending}
              onClick={apply}
            >
              {blocked ? "Разблокировать" : "Заблокировать"}
            </Button>
          </div>
        </Modal>
      )}
    </>
  );
}
