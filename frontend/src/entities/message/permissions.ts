import type { Message } from "@shared/api/types";

/**
 * Whether the viewer is offered "edit" on a message. An edit travels as
 * plaintext, so a private chat never offers it — not for an encrypted
 * message, and not for a pre-E2EE row either: the server refuses both
 * (audit A-35), and a button that always fails is worse than none.
 */
export function canEditMessage(m: Message, viewerId: string): boolean {
  return m.senderId === viewerId && !m.pending && !m.failed && !m.encrypted && m.chatType !== "private";
}
