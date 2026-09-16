import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Spinner, toast } from "@shared/ui";
import { moderationApi } from "@shared/api/endpoints";
import { ApiError } from "@shared/api/envelope";
import { moderationKeys } from "./CommunitiesTab";

// Groups deleted by moderation, each with the time left before the daily purge
// removes it for good. Restoring brings back its posts and members; its
// warnings are cut so the next one deletes it again.

const DAY = 24 * 60 * 60 * 1000;

function timeLeft(purgeAt: string): string {
  const ms = new Date(purgeAt).getTime() - Date.now();
  if (ms <= 0) return "будет удалено при следующей очистке";
  const days = Math.floor(ms / DAY);
  if (days >= 1) return `осталось ${days} дн.`;
  return `осталось ${Math.max(1, Math.ceil(ms / (60 * 60 * 1000)))} ч.`;
}

export function DeletedTab() {
  const qc = useQueryClient();
  const { data, isPending } = useQuery({ queryKey: moderationKeys.deleted, queryFn: () => moderationApi.deleted() });
  const restore = useMutation({
    mutationFn: (groupId: string) => moderationApi.restore(groupId),
    onSuccess: () => {
      toast.success("Восстановлено — основатель получит уведомление");
      void qc.invalidateQueries({ queryKey: ["admin", "moderation"] });
    },
    onError: (e) => toast.error(e instanceof ApiError ? e.message : "Не удалось восстановить"),
  });

  if (isPending) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
        <Spinner size={24} />
      </div>
    );
  }
  if (!data?.groups.length) return <p className="moderation__empty">Удалённых сообществ нет</p>;

  return (
    <ul className="moderation__list">
      {data.groups.map((g) => (
        <li key={g.id} className="moderation__item">
          <div className="moderation__row moderation__row--static">
            <Avatar name={g.name} url={g.avatarUrl} size={36} />
            <span className="moderation__who">
              <span className="moderation__name">{g.name}</span>
              <span className="moderation__sub">
                Удалено {new Date(g.deletedAt).toLocaleDateString("ru-RU")} · окончательно{" "}
                {new Date(g.purgeAt).toLocaleDateString("ru-RU")} ({timeLeft(g.purgeAt)})
              </span>
              {g.deleteReason && <span className="moderation__sanction-reason">{g.deleteReason}</span>}
            </span>
            <Button variant="secondary" loading={restore.isPending && restore.variables === g.id} onClick={() => restore.mutate(g.id)}>
              Восстановить
            </Button>
          </div>
        </li>
      ))}
    </ul>
  );
}
