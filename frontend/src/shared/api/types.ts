// Mirrors the backend DTOs (docs/spec/09-api-contracts.md and the Go
// json tags). Kept in one place so every feature shares one source of
// truth for the API shape.

export interface User {
  id: string;
  username: string;
  displayName: string;
  roleLevel: number;
  avatarUrl: string | null;
  status: "online" | "offline" | "away";
  isActive: boolean;
  lastSeen: string | null;
  createdAt: string;
}

export interface Chat {
  id: string;
  type: "private";
  otherUserId: string;
  otherUser: User | null;
  unreadCount: number;
  createdAt: string;
}

export interface Group {
  id: string;
  name: string;
  description: string | null;
  avatarUrl: string | null;
  minRoleLevel: number;
  createdAt: string;
}

export interface ReactionSummary {
  emoji: string;
  count: number;
  reacted: boolean;
}

export type ChatType = "private" | "group";

export interface Message {
  id: string;
  chatId: string;
  chatType: ChatType;
  senderId: string;
  text: string | null;
  replyTo: string | null;
  attachments: unknown[];
  reactions: ReactionSummary[];
  mentions: unknown[];
  isDeleted: boolean;
  createdAt: string;
  deletedAt: string | null;
}

export interface MessagePage {
  items: Message[];
  nextCursor?: string;
  hasMore: boolean;
}

export interface Favorite {
  chatType: ChatType;
  chatId: string;
  isPinned: boolean;
  pinnedOrder: number | null;
}

export interface Notification {
  id: string;
  type: string;
  payload: Record<string, unknown>;
  isRead: boolean;
  createdAt: string;
}

export interface Invitation {
  token: string;
  creatorId: string;
  expiresAt: string;
}

export interface AuditEntry {
  id: string;
  actorId: string | null;
  action: string;
  targetType: string | null;
  targetId: string | null;
  requestId: string | null;
  metadata: Record<string, unknown>;
  createdAt: string;
}

// Role level → human label (1 = CEO … 10 = Guest).
export const ROLE_LABELS: Record<number, string> = {
  1: "CEO",
  2: "Executive",
  3: "Director",
  4: "Senior Manager",
  5: "Manager",
  6: "Team Lead",
  7: "Senior Employee",
  8: "Employee",
  9: "Contractor",
  10: "Guest",
};

export function roleLabel(level: number): string {
  return ROLE_LABELS[level] ?? `Level ${level}`;
}
