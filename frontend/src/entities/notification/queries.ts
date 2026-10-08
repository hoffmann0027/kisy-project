import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "@shared/api/endpoints";
import type { Notification } from "@shared/api/types";
import { closeWithdrawnAnnouncementPushes } from "@shared/lib/shownNotifications";

export const notificationKeys = { list: ["notifications"] as const };

/** The announcements a list still holds — the ones whose pushes may stay. */
export function announcementIds(list: Notification[]): string[] {
  return list
    .filter((n) => n.type === "announcement" && typeof n.payload.announcementId === "string")
    .map((n) => n.payload.announcementId as string);
}

export function useNotifications() {
  return useQuery({
    queryKey: notificationKeys.list,
    queryFn: async () => {
      const res = await notificationsApi.list();
      void closeWithdrawnAnnouncementPushes(announcementIds(res.notifications));
      return res;
    },
    refetchInterval: 60_000,
  });
}

export function useMarkNotificationsRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id?: string) => notificationsApi.markRead(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: notificationKeys.list }),
  });
}
