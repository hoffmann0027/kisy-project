import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Spinner, toast } from "@shared/ui";
import { moderationApi } from "@shared/api/endpoints";
import { ApiError } from "@shared/api/envelope";
import { intlLocale, t } from "@shared/i18n";
import { moderationKeys } from "./CommunitiesTab";

// Groups deleted by moderation, each with the time left before the daily purge
// removes it for good. Restoring brings back its posts and members; its
// warnings are cut so the next one deletes it again.

const DAY = 24 * 60 * 60 * 1000;

function timeLeft(purgeAt: string): string {
  const ms = new Date(purgeAt).getTime() - Date.now();
  if (ms <= 0) return t("admin.deleted.purgeNext");
  const days = Math.floor(ms / DAY);
  if (days >= 1) return t("admin.deleted.daysLeft", { count: days });
  return t("admin.deleted.hoursLeft", { count: Math.max(1, Math.ceil(ms / (60 * 60 * 1000))) });
}

export function DeletedTab() {
  const qc = useQueryClient();
  const { data, isPending } = useQuery({ queryKey: moderationKeys.deleted, queryFn: () => moderationApi.deleted() });
  const restore = useMutation({
    mutationFn: (groupId: string) => moderationApi.restore(groupId),
    onSuccess: () => {
      toast.success(t("admin.deleted.restored"));
      void qc.invalidateQueries({ queryKey: ["admin", "moderation"] });
    },
    onError: (e) => toast.error(e instanceof ApiError ? e.message : t("admin.deleted.restoreFailed")),
  });

  if (isPending) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
        <Spinner size={24} />
      </div>
    );
  }
  if (!data?.groups.length) return <p className="moderation__empty">{t("admin.deleted.empty")}</p>;

  return (
    <ul className="moderation__list">
      {data.groups.map((g) => (
        <li key={g.id} className="moderation__item">
          <div className="moderation__row moderation__row--static">
            <Avatar name={g.name} url={g.avatarUrl} size={36} />
            <span className="moderation__who">
              <span className="moderation__name">{g.name}</span>
              <span className="moderation__sub">
                {t("admin.deleted.dates", {
                  deletedAt: new Date(g.deletedAt).toLocaleDateString(intlLocale()),
                  purgeAt: new Date(g.purgeAt).toLocaleDateString(intlLocale()),
                  left: timeLeft(g.purgeAt),
                })}
              </span>
              {g.deleteReason && <span className="moderation__sanction-reason">{g.deleteReason}</span>}
            </span>
            <Button variant="secondary" loading={restore.isPending && restore.variables === g.id} onClick={() => restore.mutate(g.id)}>
              {t("admin.deleted.restore")}
            </Button>
          </div>
        </li>
      ))}
    </ul>
  );
}
