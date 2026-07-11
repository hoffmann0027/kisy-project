import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { notifPrefsApi, type NotificationSettings } from "@shared/api/endpoints";
import type { ChatType } from "@shared/api/types";

const muteKey = ["notif-mutes"] as const;
const settingsKey = ["notif-settings"] as const;

// The actor's active mutes, as a Set of "chatType:chatId" for quick lookup.
export function useMutes() {
  const query = useQuery({
    queryKey: muteKey,
    queryFn: async () => (await notifPrefsApi.listMutes()).mutes,
  });
  const mutedSet = new Set((query.data ?? []).map((m) => `${m.chatType}:${m.chatId}`));
  return { ...query, mutedSet };
}

export function isMuted(set: Set<string>, chatType: ChatType, chatId: string): boolean {
  return set.has(`${chatType}:${chatId}`);
}

export function useMuteChat() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { chatType: ChatType; chatId: string; untilSeconds?: number; mute: boolean }) =>
      args.mute
        ? notifPrefsApi.mute(args.chatType, args.chatId, args.untilSeconds)
        : notifPrefsApi.unmute(args.chatType, args.chatId),
    onSuccess: () => qc.invalidateQueries({ queryKey: muteKey }),
  });
}

export function useNotificationSettings() {
  return useQuery({
    queryKey: settingsKey,
    queryFn: async () => (await notifPrefsApi.getSettings()).settings,
  });
}

export function useUpdateNotificationSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (s: NotificationSettings) => notifPrefsApi.updateSettings(s),
    onSuccess: (res) => qc.setQueryData(settingsKey, res.settings),
  });
}
