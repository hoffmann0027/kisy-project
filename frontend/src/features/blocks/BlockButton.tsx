import { useState } from "react";
import { Button, Modal, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useIsBlocked, useSetBlocked } from "@entities/block/queries";
import { t } from "@shared/i18n";

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
          toast.success(blocked ? t("account.blocks.unblocked") : t("account.blocks.blocked"));
          setOpen(false);
        },
        onError: () => toast.error(t("account.blocks.toggleFailed")),
      },
    );
  };

  return (
    <>
      <button
        className="conv__panel-toggle"
        title={blocked ? t("account.blocks.unblockName", { name }) : t("account.blocks.blockName", { name })}
        onClick={() => setOpen(true)}
      >
        <Icon.Ban size={20} />
      </button>

      {open && (
        <Modal open title={blocked ? t("account.blocks.unblock") : t("account.blocks.block")} onClose={() => setOpen(false)}>
          {blocked ? (
            <p className="block-dialog__text">{t("account.blocks.unblockText", { name })}</p>
          ) : (
            <>
              <p className="block-dialog__text">{t("account.blocks.blockIntro", { name })}</p>
              <ul className="block-dialog__list">
                <li>{t("account.blocks.blockNoContact")}</li>
                <li>{t("account.blocks.blockHidden")}</li>
                <li>{t("account.blocks.blockSilent")}</li>
              </ul>
              <p className="block-dialog__text">{t("account.blocks.blockUndo")}</p>
            </>
          )}
          <div className="block-dialog__actions">
            <Button variant="secondary" block onClick={() => setOpen(false)}>
              {t("account.blocks.cancel")}
            </Button>
            <Button
              variant={blocked ? "primary" : "danger"}
              block
              loading={setBlocked.isPending}
              onClick={apply}
            >
              {blocked ? t("account.blocks.unblock") : t("account.blocks.block")}
            </Button>
          </div>
        </Modal>
      )}
    </>
  );
}
