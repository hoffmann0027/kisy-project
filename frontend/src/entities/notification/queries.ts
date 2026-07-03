import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "@shared/api/endpoints";

export const notificationKeys = { list: ["notifications"] as const };

export function useNotifications() {
  return useQuery({
    queryKey: notificationKeys.list,
    queryFn: () => notificationsApi.list(),
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
