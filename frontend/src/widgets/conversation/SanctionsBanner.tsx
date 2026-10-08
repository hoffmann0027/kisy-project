import { useQuery } from "@tanstack/react-query";
import { moderationApi } from "@shared/api/endpoints";
import { intlLocale, t } from "@shared/i18n";
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
            <strong>{t("chat.sanctions.warnings", { count: data.warns.length, limit: data.warnLimit })}</strong>{" "}
            {data.warns.length + 1 >= data.warnLimit
              ? t("chat.sanctions.nextWarningDeletes")
              : t("chat.sanctions.deletedAfter", { limit: data.warnLimit })}
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
              {data.mute.expiresAt
                ? t("chat.sanctions.mutedUntil", {
                    date: new Date(data.mute.expiresAt).toLocaleString(intlLocale(), {
                      dateStyle: "short",
                      timeStyle: "short",
                    }),
                  })
                : t("chat.sanctions.mutedForever")}
            </strong>{" "}
            {t("chat.sanctions.muteExplain", { reason: data.mute.reason })}
          </div>
        )}
      </div>
    </div>
  );
}
