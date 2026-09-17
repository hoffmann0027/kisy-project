// Scheduled sending (UPD3 stage I). For E2EE private chats the text is
// encrypted at scheduling time ("path A"): the server stores only MLS
// ciphertext, and the plaintext is cached locally under sched/<id> until
// the worker delivers the real message (hydrateMessage re-keys it).
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { scheduledApi, type ScheduleMessageBody } from "@shared/api/endpoints";
import type { ChatType, ScheduledMessage } from "@shared/api/types";
import {
  cacheScheduledPlaintext,
  cachedScheduledPlaintext,
  dropScheduledPlaintext,
  e2eeSession,
} from "@entities/e2ee";
import { encryptPrivateText } from "./encryption";

export const scheduledKey = ["scheduled-messages"] as const;

export function useScheduledMessages() {
  const query = useQuery({
    queryKey: scheduledKey,
    queryFn: async () => (await scheduledApi.list()).scheduled,
  });
  return { ...query, scheduled: query.data ?? [] };
}

/** Pending scheduled messages of one chat (for the composer badge/panel). */
export function pendingForChat(list: ScheduledMessage[], chatType: ChatType, chatId: string): ScheduledMessage[] {
  return list.filter((m) => m.status === "pending" && m.chatType === chatType && m.chatId === chatId);
}

/**
 * Resolve the display text of a scheduled message: plaintext rows carry it;
 * E2EE rows read the local sched/<id> cache (null → not readable on this
 * device).
 */
export async function scheduledDisplayText(m: ScheduledMessage): Promise<string | null> {
  if (m.text != null) return m.text;
  if (!m.ciphertext) return null;
  const s = e2eeSession();
  if (!s) return null;
  return cachedScheduledPlaintext(s, m.id);
}

export interface ScheduleArgs {
  text: string;
  sendAt: Date;
  replyTo?: string;
  attachmentIds?: string[];
}

export async function scheduleMessage(
  chatType: ChatType,
  chatId: string,
  peerUserId: string | undefined,
  args: ScheduleArgs,
): Promise<ScheduledMessage> {
  const sendAt = args.sendAt.toISOString();
  if (chatType !== "private" || !args.text) {
    const body: ScheduleMessageBody = {
      chatType,
      chatId,
      text: args.text,
      replyTo: args.replyTo,
      attachmentIds: args.attachmentIds,
      sendAt,
    };
    return (await scheduledApi.schedule(body)).scheduled;
  }
  // Encrypted now, at scheduling time ("path A"); if that fails nothing is
  // scheduled — never a plaintext copy waiting on the server (audit A-10).
  const { session, body: enc } = await encryptPrivateText(chatId, peerUserId, args.text);
  const { scheduled } = await scheduledApi.schedule({
    chatType,
    chatId,
    ...enc,
    contentKind: 1,
    replyTo: args.replyTo,
    attachmentIds: args.attachmentIds,
    sendAt,
  });
  await cacheScheduledPlaintext(session, scheduled.id, args.text);
  return scheduled;
}

export function useScheduleMessage(chatType: ChatType, chatId: string, peerUserId?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: ScheduleArgs) => scheduleMessage(chatType, chatId, peerUserId, args),
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduledKey }),
  });
}

/** Change the send time of a pending scheduled message. */
export function useRescheduleMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { id: string; sendAt: Date }) =>
      scheduledApi.update(args.id, { sendAt: args.sendAt.toISOString() }),
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduledKey }),
  });
}

/**
 * Replace the text of a pending scheduled message, re-encrypting for E2EE
 * chats (the local sched cache is refreshed too).
 */
export function useEditScheduledMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (args: { scheduled: ScheduledMessage; text: string; peerUserId?: string }) => {
      const { scheduled, text, peerUserId } = args;
      if (scheduled.chatType === "private") {
        const { session, body: enc } = await encryptPrivateText(scheduled.chatId, peerUserId, text);
        const res = await scheduledApi.update(scheduled.id, { ...enc, contentKind: 1 });
        await cacheScheduledPlaintext(session, scheduled.id, text);
        return res.scheduled;
      }
      return (await scheduledApi.update(scheduled.id, { text })).scheduled;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduledKey }),
  });
}

export function useCancelScheduled() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await scheduledApi.cancel(id);
      const s = e2eeSession();
      if (s) await dropScheduledPlaintext(s, id).catch(() => {});
      return res;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduledKey }),
  });
}
