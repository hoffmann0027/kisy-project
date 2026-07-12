// Disappearing-messages setting of a chat (UPD3 stage J).
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { disappearApi, messagesApi } from "@shared/api/endpoints";
import type { ChatType } from "@shared/api/types";

const key = (chatType: ChatType, chatId: string) => ["disappearing", chatType, chatId] as const;

export function useDisappearSetting(chatType: ChatType, chatId: string) {
  return useQuery({
    queryKey: key(chatType, chatId),
    queryFn: async () => (await disappearApi.get(chatType, chatId)).disappearing,
  });
}

export function useSetDisappearing(chatType: ChatType, chatId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ttlSeconds: number | null) => disappearApi.set(chatType, chatId, ttlSeconds),
    onSuccess: (res) => qc.setQueryData(key(chatType, chatId), res.disappearing),
  });
}

/** Per-message timer (sender-only); the update lands via message.updated. */
export function useSetMessageExpiry() {
  return useMutation({
    mutationFn: (args: { messageId: string; ttlSeconds: number | null }) =>
      messagesApi.setExpiry(args.messageId, args.ttlSeconds),
  });
}

/** Human label for a TTL ("1 час", "24 часа", "7 дней", "5 мин"…). */
export function ttlLabel(seconds: number): string {
  if (seconds % 86400 === 0) {
    const d = seconds / 86400;
    return d === 1 ? "1 день" : `${d} дн.`;
  }
  if (seconds % 3600 === 0) {
    const h = seconds / 3600;
    return h === 1 ? "1 час" : `${h} ч.`;
  }
  if (seconds % 60 === 0) return `${seconds / 60} мин`;
  return `${seconds} сек`;
}
