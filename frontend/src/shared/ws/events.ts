// Server→client event names and payloads, mirroring backend/internal/ws.
import type { Message } from "@shared/api/types";

export type ServerEvent =
  | { event: "message.created"; data: Message }
  | { event: "message.deleted"; data: { chatType: string; chatId: string; messageId: string } }
  | { event: "message.read"; data: ReadData }
  | { event: "typing.started"; data: TypingData }
  | { event: "typing.stopped"; data: TypingData }
  | { event: "user.online"; data: { userId: string } }
  | { event: "user.offline"; data: { userId: string } }
  | { event: "reaction.added"; data: ReactionEvent }
  | { event: "reaction.removed"; data: ReactionEvent }
  | { event: "notification.created"; data: Record<string, unknown> }
  | { event: "error"; data: { message: string } };

export interface TypingData {
  chatType: string;
  chatId: string;
  userId: string;
}

export interface ReadData extends TypingData {
  messageId: string;
}

export interface ReactionEvent {
  chatType: string;
  chatId: string;
  messageId: string;
  userId: string;
  emoji: string;
}

// Client→server frames.
export type ClientFrame =
  | { type: "typing.start"; data: { chatType: string; chatId: string } }
  | { type: "typing.stop"; data: { chatType: string; chatId: string } }
  | { type: "presence.subscribe"; data: { userIds: string[] } }
  | { type: "read.confirmation"; data: { chatType: string; chatId: string; messageId: string } }
  | { type: "message.send"; data: { chatType: string; chatId: string; text: string; replyTo?: string } };
