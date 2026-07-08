// Server→client event names and payloads, mirroring backend/internal/ws.
import type { Message, User } from "@shared/api/types";

export type ServerEvent =
  | { event: "message.created"; data: Message }
  | { event: "message.updated"; data: Message }
  | { event: "message.deleted"; data: { chatType: string; chatId: string; messageId: string } }
  | { event: "message.read"; data: ReadData }
  | { event: "typing.started"; data: TypingData }
  | { event: "typing.stopped"; data: TypingData }
  | { event: "user.online"; data: { userId: string } }
  | { event: "user.offline"; data: { userId: string } }
  | { event: "user.updated"; data: User }
  | { event: "reaction.added"; data: ReactionEvent }
  | { event: "reaction.removed"; data: ReactionEvent }
  | { event: "notification.created"; data: Record<string, unknown> }
  | { event: "board.changed"; data: { groupId: string } }
  | { event: "group.changed"; data: { groupId: string } }
  | { event: "rating.changed"; data: Record<string, never> }
  | { event: "poll.changed"; data: Record<string, never> }
  | { event: "call.incoming"; data: CallIncomingData }
  | { event: "call.answered"; data: { callId: string; sdp: string } }
  | { event: "call.ice"; data: { callId: string; fromUserId: string; candidate: RTCIceCandidateInit } }
  | { event: "call.rejected"; data: { callId: string } }
  | { event: "call.canceled"; data: { callId: string } }
  | { event: "call.ended"; data: { callId: string; reason: string } }
  | { event: "call.busy"; data: { callId: string } }
  | { event: "call.timeout"; data: { callId: string } }
  | { event: "error"; data: { message: string } };

export interface CallIncomingData {
  callId: string;
  from: { id: string; displayName: string; avatarUrl: string | null };
  chatId: string;
  sdp: string;
}

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
  | { type: "message.send"; data: { chatType: string; chatId: string; text: string; replyTo?: string } }
  | { type: "call.invite"; data: { callId: string; toUserId: string; chatId: string; sdp: string } }
  | { type: "call.answer"; data: { callId: string; sdp: string } }
  | { type: "call.ice"; data: { callId: string; candidate: RTCIceCandidateInit } }
  | { type: "call.reject"; data: { callId: string } }
  | { type: "call.cancel"; data: { callId: string } }
  | { type: "call.hangup"; data: { callId: string } };
