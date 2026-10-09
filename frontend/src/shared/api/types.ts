// Mirrors the backend DTOs (docs/spec/09-api-contracts.md and the Go
// json tags). Kept in one place so every feature shares one source of
// truth for the API shape.
import { t } from "@shared/i18n";

/**
 * How the account came to exist. "invited" redeemed a CEO invitation and holds
 * a clearance level; "basic" registered openly and holds none.
 */
export type AccountKind = "basic" | "invited";

/**
 * The hold on a freshly self-registered account (backend: internal/quarantine).
 * Present in the /users/me response only while it lasts.
 */
export interface Quarantine {
  /** When the hold ends (ISO). */
  until: string;
  /** Hours left, rounded up — what the notices show. */
  hoursLeft: number;
  /** How many new conversations a day the account may start. */
  newChatsPerDay: number;
  /** Largest single file it may upload, in bytes. */
  maxUploadBytes: number;
}

/** На что можно пожаловаться и почему (E-02). */
export type ReportTargetKind = "user" | "message" | "post" | "community";
export type ReportReason = "spam" | "abuse" | "fraud" | "illegal" | "other";

/** Жалоба в очереди CEO. */
export interface Report {
  id: string;
  reporterId: string;
  targetKind: ReportTargetKind;
  targetId: string;
  targetOwner: string | null;
  reason: ReportReason;
  comment?: string;
  status: "open" | "resolved" | "rejected";
  createdAt: string;
  /** Сколько открытых жалоб у этого объекта всего. */
  sameTarget: number;
  /** Сколько открытых жалоб у автора объекта. */
  againstOwner: number;
  /** Текст, если сервер вообще может его прочитать. */
  content: string | null;
  /** false — содержимое есть, но оно зашифровано (личное сообщение). */
  readable: boolean;
}

export interface ReportCounts {
  open: number;
  resolved: number;
  rejected: number;
  /** Посты, скрытые из ленты жалобами до решения CEO. */
  hidden: number;
}

/** Одна запись в списке заблокированных (E-02). */
export interface BlockedUser {
  userId: string;
  createdAt: string;
  /** Профиль, если сервер смог его отдать. */
  user?: User;
}

export interface User {
  id: string;
  username: string;
  displayName: string;
  /**
   * Clearance 1–10, or null for an account that registered without an
   * invitation. Null is not "level 10": such an account stands outside the
   * role hierarchy rather than at the bottom of it, and that is what decides
   * whether levels are shown at all.
   */
  roleLevel: number | null;
  accountKind: AccountKind;
  avatarUrl: string | null;
  status: "online" | "offline" | "away";
  isActive: boolean;
  lastSeen: string | null;
  createdAt: string;
  /** When true the account must set a new password before using the app
   * (seeded bootstrap CEO). Absent/false otherwise. */
  mustChangePassword?: boolean;
  /**
   * The account's name broke the display-name rule or collided with an
   * earlier account's when the rule arrived (migration 46): the app stays
   * behind a rename screen until a new one is chosen. Present only when true.
   */
  displayNameNeedsChange?: boolean;
  /**
   * The account has not accepted the current privacy policy and community
   * rules (never, or an older version): the app stays behind the consent
   * screen until it does. Present only when true, and only on your own
   * profile (backend: internal/consent).
   */
  consentRequired?: boolean;
  /** When the CEO verified the account; null/absent when it carries no mark. */
  verifiedAt?: string | null;
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

export type JoinPolicy = "open" | "request";
export type PostPolicy = "all" | "editors";
export type GroupRole = "member" | "moderator" | "editor" | "owner";

export interface Group {
  id: string;
  name: string;
  description: string | null;
  avatarUrl: string | null;
  /**
   * Null when the group has no threshold: open to everyone, accounts outside
   * the role hierarchy included. Not 10 — that is still a threshold.
   */
  minRoleLevel: number | null;
  /** A group is a conversation; a community is a wall with a feed. */
  kind: GroupKind;
  /** Communities only: the posts appear in the shared feed. */
  isPublic: boolean;
  joinPolicy: JoinPolicy;
  postPolicy: PostPolicy;
  createdBy: string;
  createdAt: string;
  /** When the CEO verified the group or community; null when it carries no mark. */
  verifiedAt?: string | null;
}

export type GroupKind = "group" | "community";

/** A group in the "find a group" catalogue plus the actor's request status. */
export interface DirectoryGroup extends Group {
  requestStatus?: "" | "pending";
}

/** A group member with their in-group role (GET /groups/:id/members). */
export interface GroupMember {
  user: User;
  role: GroupRole;
}

/** Someone kept out of a group (GET /groups/:id/bans). */
export interface GroupBan {
  user: User;
  bannedAt: string;
}

/** The caller's own standing in a group (GET /groups/:id/me). */
export interface GroupViewer {
  member: boolean;
  role: GroupRole | "";
  canPost: boolean;
  /** Board and calendar: every member of a group, only editors of a community. */
  canUseWorkspace: boolean;
}

export type CalendarColor = "blue" | "green" | "red" | "orange" | "purple" | "teal" | "pink" | "gray";
export const CALENDAR_COLORS: CalendarColor[] = ["blue", "green", "red", "orange", "purple", "teal", "pink", "gray"];

/** A one-off group calendar event (kind "event"). */
export interface CalendarEvent {
  kind: "event";
  id: string;
  groupId: string;
  title: string;
  startsAt: string;
  endsAt: string | null;
  color: CalendarColor;
  createdBy: string;
}

/** A board card surfaced in the calendar by its due date (read-only). */
export interface CalendarCardRef {
  cardId: string;
  title: string;
  dueDate: string;
  columnId: string;
}

/** GET /groups/:id/calendar response. */
export interface CalendarMonth {
  events: CalendarEvent[];
  cards: CalendarCardRef[];
}

export interface ReactionSummary {
  emoji: string;
  count: number;
  reacted: boolean;
}

export type ChatType = "private" | "group";

export type AttachmentKind = "file" | "image" | "voice" | "video";

export interface Attachment {
  id: string;
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  isImage: boolean;
  url: string;
  kind: AttachmentKind;
  /** Playback length for voice/video. */
  durationMs?: number | null;
  /** Base64 peak envelope for voice bubbles (≤1024 bytes decoded). */
  waveform?: string | null;
  width?: number | null;
  height?: number | null;
}

/** Client-declared media properties sent with an upload. */
export interface AttachmentMeta {
  kind?: AttachmentKind;
  durationMs?: number;
  waveform?: string; // base64
  width?: number;
  height?: number;
}

/** Chunked upload session (init → chunk → complete). */
export interface UploadSession {
  id: string;
  chunkBytes: number;
  declaredBytes: number;
  expiresAt: string;
  receivedChunks: number[];
}

export interface UploadLimit {
  maxBytes: number;
  chunkBytes: number;
}

/** One entry of the chat context panel's Media/Files tabs. */
export interface ChatMediaItem {
  attachment: Attachment;
  messageId: string;
  senderId: string;
  createdAt: string;
}

export interface ChatMediaPage {
  items: ChatMediaItem[];
  nextCursor?: string;
  hasMore: boolean;
}

/** One entry of the Links tab (plaintext messages only — E2EE bodies are
 * unreadable to the server by design). */
export interface ChatLinkItem {
  url: string;
  messageId: string;
  senderId: string;
  createdAt: string;
}

export interface ChatLinkPage {
  items: ChatLinkItem[];
  nextCursor?: string;
  hasMore: boolean;
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

  /** E2EE body: base64 MLS ciphertext; the server never sees the text. */
  ciphertext?: string | null;
  /** E2EE scheme version (present on encrypted messages). */
  alg?: number | null;
  /** MLS epoch of the encrypted message. */
  epoch?: number | null;
  /** 1 text, 2 attachment, 3 system — without revealing content. */
  contentKind?: number | null;
  /** Client-only: text was decrypted locally from ciphertext. */
  encrypted?: boolean;
  /** Client-only: ciphertext could not be decrypted on this device. */
  undecryptable?: boolean;

  /** Forwarding attribution: the original author snapshot (never the source
   * chat), shown as "Переслано от …". */
  forwardedFrom?: { senderId: string; senderName: string } | null;

  /** Group messages: the author's display name (the server adds it, so a
   * member who has left still has one). */
  senderName?: string;

  /** Scheduled origin (stage I): id of the scheduled_messages row this
   * message was born from — lets the sender restore its plaintext cache. */
  scheduledId?: string | null;

  /** Disappearing (stage J): when this message self-destructs (hard-deleted
   * server-side; the client purges its local plaintext cache too). */
  expiresAt?: string | null;

  /** Threads (stage K, groups): set on replies / roots respectively. */
  threadRootId?: string | null;
  threadReplyCount?: number;
  threadLastReplyAt?: string | null;
}

/** A chat's disappearing-messages default (stage J). */
export interface DisappearSetting {
  ttlSeconds: number | null;
  setBy?: string | null;
  updatedAt?: string | null;
}

/** A scheduled (delayed-send) message — a frozen send-body snapshot. */
export interface ScheduledMessage {
  id: string;
  chatType: ChatType;
  chatId: string;
  text: string | null;
  ciphertext?: string | null;
  alg?: number | null;
  epoch?: number | null;
  contentKind?: number | null;
  replyTo: string | null;
  attachmentIds: string[];
  sendAt: string;
  status: "pending" | "sent" | "canceled";
  sentMessageId?: string | null;
  createdAt: string;
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

/** The admin overview ("Обзор"), GET /admin/dashboard — real data only. */
export interface DashboardOverview {
  kpi: {
    usersTotal: number;
    usersNew24h: number;
    usersNew7d: number;
    active24h: number;
    messages24h: number;
    communities: number;
    groups: number;
  };
  /** Registrations per day for 90 days, with the running total. */
  growth: { day: string; registrations: number; total: number }[];
  inbox: { openReports: number; unansweredFeedback: number; pendingJoinRequests: number };
  activity: { action: string; actorName: string | null; targetType: string | null; createdAt: string }[];
  reports: { id: string; targetKind: string; reason: string; severity: "high" | "medium" | "low"; createdAt: string }[];
  system: {
    version: string;
    startedAt: string;
    checks: { name: string; state: "ok" | "down" | "off"; latencyMs?: number; detail?: string }[];
  };
  limits: {
    database: { usedBytes: number; limitBytes: number };
    filesInDatabase: number;
    redis: { usedBytes: number; limitBytes: number } | null;
  };
}

/** An announced version of the app ("Обновления"). */
export interface AppRelease {
  id: string;
  version: string;
  notes: string;
  downloadUrl?: string;
  recipientCount: number;
  createdAt: string;
}

/** How many accounts ran one Android build in the last 30 days. */
export interface AppBuildCount {
  version: string;
  build: number;
  users: number;
}

/**
 * Search and filters of the admin "Пользователи" tab. role: "basic" or a level
 * "1".."10"; empty fields filter nothing.
 */
export interface AdminUserFilter {
  q?: string;
  role?: string;
  status?: "active" | "inactive";
}

/** Who an announcement goes to. */
export type AnnouncementAudience = "all" | "basic" | "levels" | "user";

/** The author as recipients see them: an announcement is always from a person. */
export interface AnnouncementAuthor {
  id: string;
  displayName: string;
  /** 0 for a basic account. */
  roleLevel: number;
}

/** A sent announcement, as its author (or the CEO) sees it. */
export interface Announcement {
  id: string;
  author: AnnouncementAuthor;
  audience: AnnouncementAudience;
  levels?: number[];
  targetUserId?: string;
  targetName?: string;
  title: string;
  body: string;
  recipientCount: number;
  createdAt: string;
  revokedAt?: string;
}

export interface SendAnnouncementInput {
  audience: AnnouncementAudience;
  levels?: number[];
  userId?: string;
  title: string;
  body: string;
}

export interface Invitation {
  token: string;
  creatorId: string;
  expiresAt: string;
}

export interface FeedbackAuthor {
  id: string;
  displayName: string;
  username?: string;
  avatarUrl: string | null;
  /** Null for an account outside the hierarchy. */
  roleLevel: number | null;
}

/** Leadership's answer to an entry. */
export interface FeedbackReply {
  body: string;
  at: string;
  /** Null when whoever answered has since been removed. */
  by: FeedbackAuthor | null;
}

export interface FeedbackItem {
  id: string;
  body: string;
  author: FeedbackAuthor;
  createdAt: string;
  reply: FeedbackReply | null;
}

/** "mine": the caller's own entries; "inbox": unanswered ones, levels 1-3 only. */
export type FeedbackScope = "mine" | "inbox";

export interface FeedbackPage {
  items: FeedbackItem[];
  nextCursor?: string | null;
  hasMore: boolean;
}

export interface IceServer {
  urls: string[];
  username?: string;
  credential?: string;
}

export interface IceConfig {
  iceServers: IceServer[];
}

export interface CallLogItem {
  id: string;
  direction: "incoming" | "outgoing";
  status: "completed" | "missed" | "rejected" | "canceled" | "failed";
  peer: { id: string; displayName: string; avatarUrl: string | null };
  chatId: string;
  startedAt: string;
  answeredAt: string | null;
  endedAt: string | null;
  durationSeconds: number;
}

export interface PollVoter {
  id: string;
  displayName: string;
  avatarUrl: string | null;
}

export interface PollOption {
  id: string;
  body: string;
  votes: number;
  voters: PollVoter[];
}

export interface Poll {
  id: string;
  question: string;
  status: "open" | "closed";
  options: PollOption[];
  totalVotes: number;
  myOptionId: string | null;
  createdAt: string;
  closedAt: string | null;
}

export interface LevelCondition {
  targetLevel: number;
  body: string;
  updatedAt: string;
}

export interface Note {
  id: string;
  text: string | null;
  hasFile: boolean;
  fileName: string | null;
  fileType: string | null;
  fileSize: number;
  fileUrl: string | null;
  createdAt: string;
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
  minLevel: number;
  status: "active" | "done";
  createdBy: string;
  /** The project's ledger: income and expense apart, profit their difference (euro cents). */
  totalIncomeKopecks: number;
  totalExpenseKopecks: number;
  totalProfitKopecks: number;
  tasks: RatingTask[];
  /** Who answers for the project and records its money; changed by its creator (levels 1–4) or the CEO. */
  members: RatingAssignee[];
  createdAt: string;
  /** When anything last happened: a task moved, money recorded, the project completed. */
  updatedAt: string;
  completedAt: string | null;
}

export interface RatingBoard {
  projects: RatingProject[];
}

export interface RatingProjectMoney {
  projectId: string;
  title: string;
  incomeKopecks: number;
  expenseKopecks: number;
  profitKopecks: number;
}
export interface RatingMonth {
  /** "YYYY-MM" */
  month: string;
  incomeKopecks: number;
  expenseKopecks: number;
  profitKopecks: number;
}
/** One recorded income or expense (GET /rating/analytics, the latest few). */
export interface RatingLedgerEntry {
  id: string;
  projectId: string;
  projectTitle: string;
  incomeKopecks: number;
  expenseKopecks: number;
  note: string | null;
  authorName: string;
  createdAt: string;
}
export interface RatingAnalytics {
  perProject: RatingProjectMoney[];
  monthly: RatingMonth[];
  recent: RatingLedgerEntry[];
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

/**
 * The label under someone's name. Empty for an account with no level: there is
 * no such thing as an unranked rank, and inventing one ("Гость", "Level 0")
 * would put it in a hierarchy it does not belong to.
 */
export function roleLabel(level: number | null | undefined): string {
  if (level == null) return "";
  return ROLE_LABELS[level] ?? `Level ${level}`;
}

/**
 * The line under someone's name in a list: "@name · Руководитель", or just
 * "@name" for an account with no level. Written once because the separator is
 * the trap — "@name · " with nothing after it is how a missing level shows up.
 */
export function userSubtitle(user: Pick<User, "username" | "roleLevel">): string {
  const label = roleLabel(user.roleLevel);
  return label ? `@${user.username} · ${label}` : `@${user.username}`;
}

// --- Community posts and the feed (stage 2, step 3) ---

export interface PostMedia {
  id: string;
  kind: "image" | "video" | "audio" | "file";
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  url: string;
  position: number;
}

export interface PostReaction {
  emoji: string;
  count: number;
  /** Whether the signed-in user is one of the people who chose it. */
  mine: boolean;
}

/**
 * Where a post came from, carried on every post.
 *
 * Without this a reader sees a post in the feed and cannot tell whose wall it
 * is, let alone get to it — which is what makes "open" and "join" possible
 * straight from the card.
 */
export interface PostCommunity {
  id: string;
  name: string;
  avatarUrl: string | null;
  isMember: boolean;
  /** "open" joins instantly; "request" sends an application. */
  joinPolicy: JoinPolicy;
  /** The community carries the verification mark. */
  verified: boolean;
}

export interface Post {
  id: string;
  text: string;
  createdAt: string;
  editedAt: string | null;
  /** A post speaks as its community; the editor who wrote it is not sent. */
  community: PostCommunity;
  media: PostMedia[];
  reactions: PostReaction[];
  canDelete: boolean;
}

export interface PostPage {
  posts: Post[];
  /** Empty when there is nothing more to load. */
  nextCursor: string;
}

/** Feed ordering: the ranking, or plain reverse chronological. */
export type FeedSort = "popular" | "new";

/**
 * The line under a group's name in a list.
 *
 * Three facts, and each may be absent: a community is not a group, and a
 * group with no threshold has no level to name — "от  и выше" with a hole in
 * the middle is what happens when that is forgotten.
 */
export function groupSubtitle(group: Pick<Group, "kind" | "minRoleLevel" | "isPublic">): string {
  const noun =
    group.kind === "community"
      ? t(group.isPublic ? "common.group.openCommunity" : "common.group.community")
      : t("common.group.group");
  if (group.minRoleLevel === null) return noun;
  return t("common.group.fromRole", { kind: noun, role: roleLabel(group.minRoleLevel) });
}

// --- CEO moderation of groups and communities (migration 48) ---

export type SanctionKind = "warn" | "mute" | "delete";
/** How long a mute lasts; "forever" is indefinite. */
export type MuteDuration = "1d" | "7d" | "30d" | "forever";

export interface Sanction {
  id: string;
  groupId: string;
  kind: SanctionKind;
  /** The CEO's comment — always present, shown to the group's founder and editors. */
  reason: string;
  issuedBy: string;
  issuedAt: string;
  /** Mutes only; null is indefinite. */
  expiresAt: string | null;
  revokedAt: string | null;
  revokedBy: string | null;
  revokeNote: string | null;
}

/** A row of the CEO's communities list. */
export interface ModeratedGroup {
  id: string;
  name: string;
  kind: GroupKind;
  avatarUrl: string | null;
  isPublic: boolean;
  verifiedAt: string | null;
  memberCount: number;
  activeWarns: number;
  warnLimit: number;
  muted: boolean;
  /** Null while muted means indefinitely. */
  mutedUntil: string | null;
}

/** A group deleted by moderation, still restorable until purgeAt. */
export interface DeletedGroup {
  id: string;
  name: string;
  kind: GroupKind;
  avatarUrl: string | null;
  deletedAt: string;
  purgeAt: string;
  deleteReason: string;
}

/** What a group's own editors see in the banner. */
export interface ActiveSanctions {
  warns: Sanction[];
  warnLimit: number;
  mute: Sanction | null;
}

export interface SanctionOutcome {
  sanction: Sanction;
  activeWarns: number;
  deleted: boolean;
}
