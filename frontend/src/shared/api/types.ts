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
  /** When the counterpart last read this chat (for read receipts); null if never. */
  otherLastReadAt: string | null;
  createdAt: string;
}

export interface Group {
  id: string;
  name: string;
  description: string | null;
  avatarUrl: string | null;
  minRoleLevel: number;
  createdBy: string;
  createdAt: string;
}

export interface ReactionSummary {
  emoji: string;
  count: number;
  reacted: boolean;
}

export type ChatType = "private" | "group";

export interface Attachment {
  id: string;
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  isImage: boolean;
  url: string;
}

export interface Message {
  id: string;
  chatId: string;
  chatType: ChatType;
  senderId: string;
  text: string | null;
  replyTo: string | null;
  attachments: Attachment[];
  reactions: ReactionSummary[];
  mentions: unknown[];
  isDeleted: boolean;
  createdAt: string;
  deletedAt: string | null;
  /** When the message was last edited (null = never). */
  editedAt: string | null;
  /** When the message was pinned (null = not pinned). */
  pinnedAt: string | null;
  /** For own group messages: how many recipients read it, of how many. */
  readCount: number | null;
  readTotal: number | null;
  /** Client-only: true while an optimistically-sent message awaits server ack. */
  pending?: boolean;
  /** Client-only: true if the optimistic send failed. */
  failed?: boolean;
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

export interface FeedbackAuthor {
  id: string;
  displayName: string;
  username: string;
  avatarUrl: string | null;
  roleLevel: number;
}

export interface FeedbackItem {
  id: string;
  body: string;
  author: FeedbackAuthor;
  createdAt: string;
}

export interface FeedbackPage {
  items: FeedbackItem[];
  nextCursor?: string | null;
  hasMore: boolean;
}

export interface BoardCard {
  id: string;
  columnId: string;
  title: string;
  description: string | null;
  position: number;
  assigneeId: string | null;
  label: string | null;
  dueDate: string | null;
  createdBy: string;
  createdAt: string;
}

export interface SearchResult {
  messageId: string;
  chatType: ChatType;
  chatId: string;
  senderId: string;
  senderName: string;
  text: string;
  createdAt: string;
}

export type RatingDifficulty = "easy" | "medium" | "hard";
export type RatingStatus = "backlog" | "in_progress" | "done";

export interface RatingAssignee {
  id: string;
  displayName: string;
  avatarUrl: string | null;
}

export interface RatingTask {
  id: string;
  projectId: string;
  projectTitle: string;
  title: string;
  assignee: RatingAssignee | null;
  progress: number;
  status: RatingStatus;
  totalProfitKopecks: number;
  createdAt: string;
}

export interface RatingProject {
  id: string;
  title: string;
  description: string | null;
  difficulty: RatingDifficulty;
  status: "active" | "done";
  createdBy: string;
  totalProfitKopecks: number;
  tasks: RatingTask[];
  createdAt: string;
}

export interface RatingBoard {
  projects: RatingProject[];
}

export interface RatingAnalytics {
  perProject: { projectId: string; title: string; profitKopecks: number }[];
  monthly: { month: string; profitKopecks: number }[];
}

export interface BoardColumn {
  id: string;
  title: string;
  position: number;
  cards: BoardCard[];
}

export interface Board {
  id: string;
  groupId: string;
  title: string;
  createdBy: string;
  columns: BoardColumn[];
}

export interface CardInput {
  title: string;
  description?: string | null;
  assigneeId?: string | null;
  label?: string | null;
  dueDate?: string | null;
}

// Card label palette (key → display color), used by the board UI.
export const CARD_LABELS: Record<string, string> = {
  blue: "#0a84ff",
  green: "#32d74b",
  yellow: "#ffd60a",
  red: "#ff453a",
  purple: "#bf5af2",
  gray: "#8e8e93",
};

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
