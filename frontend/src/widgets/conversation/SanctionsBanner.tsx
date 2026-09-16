import { useQuery } from "@tanstack/react-query";
import { moderationApi } from "@shared/api/endpoints";
import { Icon } from "@shared/ui/icons";

// The CEO's live sanctions on a group, shown to the people who run it — its
// founder and editors — at the top of the group. Readers never see it (the
// server refuses them, and this is not asked for on their behalf).

export const sanctionKeys = { active: (groupId: string) => ["group-sanctions", groupId] as const };

export function SanctionsBanner({ groupId, runsGroup }: { groupId: string; runsGroup: boolean }) {
  const { data } = useQuery({
    queryKey: sanctionKeys.active(groupId),
    queryFn: () => moderationApi.active(groupId),
    enabled: runsGroup,
    retry: false,
  });
  if (!runsGroup || !data || (data.warns.length === 0 && !data.mute)) return null;

  return (
    <div className="sanctions-banner" role="status">
      <Icon.Shield size={18} />
      <div className="sanctions-banner__body">
        {data.warns.length > 0 && (
          <div>
            <strong>
              Предупреждения: {data.warns.length} из {data.warnLimit}.
            </strong>{" "}
            {data.warns.length + 1 >= data.warnLimit
              ? "Следующее предупреждение удалит сообщество."
              : `После ${data.warnLimit}-го сообщество будет удалено.`}
            <ul className="sanctions-banner__reasons">
              {data.warns.map((w) => (
                <li key={w.id}>{w.reason}</li>
              ))}
            </ul>
          </div>
        )}
        {data.mute && (
          <div>
            <strong>
              Скрыто из ленты{" "}
              {data.mute.expiresAt
                ? `до ${new Date(data.mute.expiresAt).toLocaleString("ru-RU", { dateStyle: "short", timeStyle: "short" })}`
                : "бессрочно"}
              .
            </strong>{" "}
            Публиковать можно, но посты не попадут в общую ленту. Причина: {data.mute.reason}
          </div>
        )}
      </div>
    </div>
  );
}
