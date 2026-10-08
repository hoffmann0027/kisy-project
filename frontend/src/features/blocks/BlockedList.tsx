import { Avatar, Button, toast } from "@shared/ui";
import { useBlocks, useSetBlocked } from "@entities/block/queries";
import { t } from "@shared/i18n";

// The list of people this account has blocked, in profile settings. Empty for
// most people — so it says nothing at all rather than showing an empty box.

export function BlockedList() {
  const blocks = useBlocks();
  const setBlocked = useSetBlocked();

  if (blocks.length === 0) return null;

  return (
    <div className="profile-section">
      <div className="profile-section__label">{t("account.blocks.listTitle", { count: blocks.length })}</div>
      <ul className="blocked-list">
        {blocks.map((b) => {
          const name = b.user?.displayName ?? b.user?.username ?? t("account.blocks.unknownUser");
          return (
            <li key={b.userId} className="blocked-list__row">
              <Avatar name={name} url={b.user?.avatarUrl ?? null} size={32} />
              <span className="blocked-list__name">{name}</span>
              <Button
                variant="secondary"
                onClick={() =>
                  setBlocked.mutate(
                    { userId: b.userId, blocked: false },
                    {
                      onSuccess: () => toast.success(t("account.blocks.unblocked")),
                      onError: () => toast.error(t("account.blocks.unblockFailed")),
                    },
                  )
                }
              >
                {t("account.blocks.unblock")}
              </Button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
