import { apiClient } from "./client";
import type {
  Attachment,
  AttachmentMeta,
  AuditEntry,
  Board,
  CardInput,
  Chat,
  ChatType,
  Favorite,
  FeedbackItem,
  FeedbackPage,
  CallLogItem,
  ChatLinkPage,
  ChatMediaPage,
  Group,
  IceConfig,
  Invitation,
  LevelCondition,
  Message,
  MessagePage,
  Note,
  Notification,
  Poll,
  RatingAnalytics,
  RatingBoard,
  SearchResult,
  UploadLimit,
  UploadSession,
  User,
} from "./types";

export const authApi = {
  login: (username: string, password: string) =>
    apiClient.post<{ user: User }>("/auth/login", { username, password }),
  register: (inviteToken: string, username: string, password: string) =>
    apiClient.post<{ user: User }>("/auth/register", { inviteToken, username, password }),
  logout: () => apiClient.post<{ loggedOut: boolean }>("/auth/logout"),
  logoutAll: () => apiClient.post<{ revokedSessions: number }>("/auth/logout-all"),
  refresh: () => apiClient.post<{ accessExpiresAt: string }>("/auth/refresh"),
  changePassword: (currentPassword: string, newPassword: string) =>
    apiClient.post<{ passwordChanged: boolean }>("/auth/password", { currentPassword, newPassword }),
};

export const usersApi = {
  me: () => apiClient.get<{ user: User }>("/users/me"),
  updateUsername: (username: string) => apiClient.patch<{ user: User }>("/users/me", { username }),
  updateProfile: (fields: { displayName?: string; username?: string }) =>
    apiClient.patch<{ user: User }>("/users/me", fields),
  uploadAvatar: (blob: Blob) => apiClient.postBlob<{ user: User }>("/users/me/avatar", blob),
  directory: (search = "", limit = 25) => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (search) params.set("search", search);
    return apiClient.get<{ users: User[] }>(`/users/directory?${params.toString()}`);
  },
};

export const invitesApi = {
  create: () => apiClient.post<Invitation>("/invites"),
};

// SendMessageBody carries either plaintext (legacy) or E2EE ciphertext.
export interface SendMessageBody {
  text?: string;
  replyTo?: string;
  attachmentIds?: string[];
  /** Base64 MLS ciphertext — mutually exclusive with text. */
  ciphertext?: string;
  alg?: number;
  epoch?: number;
  contentKind?: number;
  /** Forwarded-from attribution for a client-side (E2EE) forward. */
  forwardedFromSenderId?: string;
  forwardedFromSenderName?: string;
}

export interface E2EEDeviceDTO {
  id: string;
  userId: string;
  name: string;
  ed25519Pub: string; // base64
  signedBy: string | null;
  signature?: string | null;
  createdAt: string;
  revokedAt: string | null;
}

export interface E2EEGroupMessageDTO {
  id: string;
  chatType: ChatType;
  chatId: string;
  kind: number; // 1 welcome, 2 commit, 3 proposal
  senderDevice: string | null;
  recipientDevice?: string | null;
  payload: string; // base64
  epoch: number | null;
  createdAt: string;
}

// E2EE key directory + MLS handshake mailbox (docs/e2ee-design.md §5).
export const e2eeApi = {
  registerDevice: (device: {
    deviceId: string;
    name: string;
    ed25519Pub: string;
    signedBy?: string;
    signature?: string;
  }) => apiClient.post<{ device: E2EEDeviceDTO }>("/e2ee/devices", device),
  listDevices: (userId: string) =>
    apiClient.get<{ devices: E2EEDeviceDTO[] }>(`/e2ee/users/${userId}/devices`),
  uploadKeyPackages: (deviceId: string, keyPackages: string[]) =>
    apiClient.post<{ uploaded: number }>("/e2ee/key-packages", { deviceId, keyPackages }),
  countKeyPackages: (deviceId: string) =>
    apiClient.get<{ available: number }>(`/e2ee/key-packages/count?deviceId=${deviceId}`),
  claimKeyPackages: (userId: string, excludeDevice?: string) => {
    const q = excludeDevice ? `?excludeDevice=${excludeDevice}` : "";
    return apiClient.post<{ keyPackages: { deviceId: string; keyPackage: string }[] }>(
      `/e2ee/users/${userId}/key-packages/claim${q}`,
    );
  },
  publishHandshake: (body: {
    chatType: ChatType;
    chatId: string;
    kind: "welcome" | "commit" | "proposal";
    senderDevice: string;
    payload: string;
    epoch?: number;
    recipients?: Record<string, string>; // deviceId → userId
  }) => apiClient.post<{ published: boolean }>("/e2ee/handshake", body),
  listHandshake: (chatType: ChatType, chatId: string, afterId?: string) => {
    const q = afterId ? `?afterId=${afterId}` : "";
    return apiClient.get<{ messages: E2EEGroupMessageDTO[] }>(`/e2ee/handshake/${chatType}/${chatId}${q}`);
  },
  listWelcomes: (deviceId: string) =>
    apiClient.get<{ welcomes: E2EEGroupMessageDTO[] }>(`/e2ee/welcomes?deviceId=${deviceId}`),
  ackWelcome: (welcomeId: string, deviceId: string) =>
    apiClient.post<{ acked: boolean }>(`/e2ee/welcomes/${welcomeId}/ack?deviceId=${deviceId}`),
};

// Aggregated shared content of one chat (context panel tabs, stage C).
export const chatMediaApi = {
  list: (chatType: ChatType, chatId: string, kind: "media" | "files", cursor?: string, limit = 40) => {
    const params = new URLSearchParams({ chatType, chatId, kind, limit: String(limit) });
    if (cursor) params.set("cursor", cursor);
    return apiClient.get<ChatMediaPage>(`/chats/media?${params.toString()}`);
  },
  links: (chatType: ChatType, chatId: string, cursor?: string, limit = 40) => {
    const params = new URLSearchParams({ chatType, chatId, kind: "links", limit: String(limit) });
    if (cursor) params.set("cursor", cursor);
    return apiClient.get<ChatLinkPage>(`/chats/media?${params.toString()}`);
  },
};

export type GroupNotifyMode = "all" | "mentions_only" | "none";

export interface NotificationSettings {
  sound: boolean;
  preview: boolean;
  groupMode: GroupNotifyMode;
}

export interface ChatMute {
  chatType: ChatType;
  chatId: string;
  mutedUntil: string | null;
}

// Mute + notification settings (stage G).
export const notifPrefsApi = {
  mute: (chatType: ChatType, chatId: string, untilSeconds?: number) =>
    apiClient.put<{ muted: boolean; mutedUntil: string | null }>(
      `/chats/${chatType}/${chatId}/mute`,
      untilSeconds ? { untilSeconds } : {},
    ),
  unmute: (chatType: ChatType, chatId: string) =>
    apiClient.del<{ muted: boolean }>(`/chats/${chatType}/${chatId}/mute`),
  listMutes: () => apiClient.get<{ mutes: ChatMute[] }>("/settings/mutes"),
  getSettings: () => apiClient.get<{ settings: NotificationSettings }>("/settings/notifications"),
  updateSettings: (s: NotificationSettings) =>
    apiClient.put<{ settings: NotificationSettings }>("/settings/notifications", s),
};

// Chat folders + archive (UPD3 stage H). Both are personal metadata over
// chat references; the server never lets a folder reveal an inaccessible
// chat (masked 404 on add/archive).
export interface ChatFolderItem {
  chatType: ChatType;
  chatId: string;
}

export interface ChatFolder {
  id: string;
  name: string;
  position: number;
  items: ChatFolderItem[];
}

export interface ArchivedChat {
  chatType: ChatType;
  chatId: string;
  archivedAt: string;
}

export const chatFoldersApi = {
  list: () => apiClient.get<{ folders: ChatFolder[] }>("/folders"),
  create: (name: string) => apiClient.post<{ folder: ChatFolder }>("/folders", { name }),
  rename: (id: string, name: string) => apiClient.patch<{ ok: boolean }>(`/folders/${id}`, { name }),
  remove: (id: string) => apiClient.del<{ ok: boolean }>(`/folders/${id}`),
  reorder: (folderIds: string[]) => apiClient.put<{ ok: boolean }>("/folders/order", { folderIds }),
  addItem: (id: string, chatType: ChatType, chatId: string) =>
    apiClient.post<{ ok: boolean }>(`/folders/${id}/items`, { chatType, chatId }),
  removeItem: (id: string, chatType: ChatType, chatId: string) =>
    apiClient.del<{ ok: boolean }>(`/folders/${id}/items`, { chatType, chatId }),
  archive: (chatType: ChatType, chatId: string) =>
    apiClient.put<{ archived: boolean }>(`/chats/${chatType}/${chatId}/archive`, {}),
  unarchive: (chatType: ChatType, chatId: string) =>
    apiClient.del<{ archived: boolean }>(`/chats/${chatType}/${chatId}/archive`),
  listArchived: () => apiClient.get<{ archived: ArchivedChat[] }>("/settings/archived"),
};

export interface LinkPreview {
  url: string;
  title: string;
  description: string;
  imageUrl: string;
  siteName: string;
}

// Link previews (stage E): the server fetches OpenGraph metadata behind an
// SSRF guard; the image is proxied same-origin so strict CSP holds.
export const linkPreviewApi = {
  fetch: (url: string) => apiClient.post<{ preview: LinkPreview }>("/link-preview", { url }),
  imageProxyUrl: (imageUrl: string) => `/api/v1/link-preview/image?url=${encodeURIComponent(imageUrl)}`,
};

export const chatsApi = {
  list: () => apiClient.get<{ chats: Chat[] }>("/chats"),
  open: (userId: string) => apiClient.post<{ chat: Chat }>("/chats", { userId }),
  get: (chatId: string) => apiClient.get<{ chat: Chat }>(`/chats/${chatId}`),
};

export const groupsApi = {
  list: () => apiClient.get<{ groups: Group[] }>("/groups"),
  create: (name: string, minRoleLevel: number, description?: string) =>
    apiClient.post<{ group: Group }>("/groups", { name, minRoleLevel, description }),
  get: (groupId: string) => apiClient.get<{ group: Group }>(`/groups/${groupId}`),
  remove: (groupId: string) => apiClient.del<{ deleted: boolean }>(`/groups/${groupId}`),
  members: (groupId: string) => apiClient.get<{ members: User[] }>(`/groups/${groupId}/members`),
  addMember: (groupId: string, userId: string) =>
    apiClient.post<{ added: boolean }>(`/groups/${groupId}/members`, { userId }),
  uploadAvatar: (groupId: string, blob: Blob) =>
    apiClient.postBlob<{ group: Group }>(`/groups/${groupId}/avatar`, blob),
};

export const boardsApi = {
  get: (groupId: string) => apiClient.get<{ board: Board }>(`/groups/${groupId}/board`),
  create: (groupId: string, title: string) =>
    apiClient.post<{ board: Board }>(`/groups/${groupId}/board`, { title }),
  addColumn: (boardId: string, title: string) =>
    apiClient.post<{ ok: boolean }>(`/boards/${boardId}/columns`, { title }),
  renameColumn: (columnId: string, title: string) =>
    apiClient.patch<{ ok: boolean }>(`/boards/columns/${columnId}`, { title }),
  deleteColumn: (columnId: string) => apiClient.del<{ ok: boolean }>(`/boards/columns/${columnId}`),
  createCard: (columnId: string, input: CardInput) =>
    apiClient.post<{ card: unknown }>(`/boards/columns/${columnId}/cards`, input),
  updateCard: (cardId: string, input: CardInput) =>
    apiClient.patch<{ ok: boolean }>(`/boards/cards/${cardId}`, input),
  moveCard: (cardId: string, columnId: string, index: number) =>
    apiClient.post<{ ok: boolean }>(`/boards/cards/${cardId}/move`, { columnId, index }),
  deleteCard: (cardId: string) => apiClient.del<{ ok: boolean }>(`/boards/cards/${cardId}`),
};

export const messagesApi = {
  list: (chatType: ChatType, chatId: string, cursor?: string, limit = 50) => {
    const params = new URLSearchParams({ chatType, chatId, limit: String(limit) });
    if (cursor) params.set("cursor", cursor);
    return apiClient.get<MessagePage>(`/messages?${params.toString()}`);
  },
  send: (chatType: ChatType, chatId: string, body: SendMessageBody) =>
    apiClient.post<{ message: Message }>("/messages", { chatType, chatId, ...body }),
  // Server-side forward of plaintext messages into a target chat. E2EE
  // messages are forwarded client-side via send() with forwardedFrom* set.
  forward: (sourceMessageIds: string[], targetChatType: ChatType, targetChatId: string) =>
    apiClient.post<{ messages: Message[] }>("/messages/forward", {
      sourceMessageIds,
      targetChatType,
      targetChatId,
    }),
  edit: (messageId: string, text: string) =>
    apiClient.patch<{ message: Message }>(`/messages/${messageId}`, { text }),
  remove: (messageId: string) => apiClient.del<{ deleted: boolean }>(`/messages/${messageId}`),
  pin: (messageId: string) => apiClient.post<{ message: Message }>(`/messages/${messageId}/pin`),
  unpin: (messageId: string) => apiClient.post<{ message: Message }>(`/messages/${messageId}/unpin`),
  listPinned: (chatType: ChatType, chatId: string) =>
    apiClient.get<{ pinned: Message[] }>(`/messages/pinned?chatType=${chatType}&chatId=${chatId}`),
  addReaction: (messageId: string, emoji: string) =>
    apiClient.post<{ ok: boolean }>(`/messages/${messageId}/reactions`, { emoji }),
  removeReaction: (messageId: string, emoji: string) =>
    apiClient.del<{ ok: boolean }>(`/messages/${messageId}/reactions`, { emoji }),
  markRead: (chatType: ChatType, chatId: string, messageId: string) =>
    apiClient.post<{ ok: boolean }>("/read", { chatType, chatId, messageId }),
};

// metaHeaders encodes AttachmentMeta for the single-shot raw-body upload.
function metaHeaders(meta?: AttachmentMeta): Record<string, string> {
  if (!meta) return {};
  const h: Record<string, string> = {};
  if (meta.kind) h["X-Attachment-Kind"] = meta.kind;
  if (meta.waveform) h["X-Attachment-Waveform"] = meta.waveform;
  if (meta.durationMs !== undefined) h["X-Attachment-Duration-Ms"] = String(meta.durationMs);
  if (meta.width !== undefined) h["X-Attachment-Width"] = String(meta.width);
  if (meta.height !== undefined) h["X-Attachment-Height"] = String(meta.height);
  return h;
}

export const attachmentsApi = {
  upload: (file: File, meta?: AttachmentMeta, signal?: AbortSignal) =>
    apiClient.uploadFile<{ attachment: Attachment }>("/attachments", file, metaHeaders(meta), signal),
  limit: () => apiClient.get<UploadLimit>("/attachments/limit"),
  initUpload: (fileName: string, sizeBytes: number, meta?: AttachmentMeta) =>
    apiClient.post<{ upload: UploadSession }>("/attachments/init", { fileName, sizeBytes, ...meta }),
  uploadStatus: (id: string) => apiClient.get<{ upload: UploadSession }>(`/attachments/${id}/upload-status`),
  putChunk: (id: string, index: number, chunk: Blob, signal?: AbortSignal) =>
    apiClient.putBlob<{ stored: boolean; index: number }>(`/attachments/${id}/chunk?index=${index}`, chunk, signal),
  completeUpload: (id: string) =>
    apiClient.post<{ attachment: Attachment }>(`/attachments/${id}/complete`),
};

export const pushApi = {
  vapidKey: () => apiClient.get<{ publicKey: string; enabled: boolean }>("/push/vapid-public-key"),
  subscribe: (sub: { endpoint: string; keys: { p256dh: string; auth: string } }) =>
    apiClient.post<{ subscribed: boolean }>("/push/subscribe", sub),
  unsubscribe: (endpoint: string) => apiClient.post<{ unsubscribed: boolean }>("/push/unsubscribe", { endpoint }),
};

export const favoritesApi = {
  list: () => apiClient.get<{ favorites: Favorite[] }>("/favorites"),
  set: (fav: Favorite) => apiClient.put<{ ok: boolean }>("/favorites", fav),
  remove: (chatType: ChatType, chatId: string) =>
    apiClient.del<{ ok: boolean }>("/favorites", { chatType, chatId }),
};

export const notificationsApi = {
  list: (limit = 50) =>
    apiClient.get<{ notifications: Notification[]; unreadCount: number }>(`/notifications?limit=${limit}`),
  markRead: (id?: string) => apiClient.post<{ ok: boolean }>("/notifications/read", id ? { id } : {}),
};

export const feedbackApi = {
  list: (cursor?: string, limit = 20) => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (cursor) params.set("cursor", cursor);
    return apiClient.get<FeedbackPage>(`/feedback?${params.toString()}`);
  },
  create: (body: string) => apiClient.post<{ feedback: FeedbackItem }>("/feedback", { body }),
  remove: (id: string) => apiClient.del<{ deleted: boolean }>(`/feedback/${id}`),
};

export const searchApi = {
  messages: (q: string, limit = 25) =>
    apiClient.get<{ results: SearchResult[] }>(`/search?q=${encodeURIComponent(q)}&limit=${limit}`),
};

export const ratingApi = {
  board: () => apiClient.get<RatingBoard>("/rating/board"),
  analytics: () => apiClient.get<RatingAnalytics>("/rating/analytics"),
  createProject: (title: string, minLevel: number, description?: string) =>
    apiClient.post<{ id: string }>("/rating/projects", { title, minLevel, description }),
  setProjectLevel: (id: string, minLevel: number) =>
    apiClient.patch<{ ok: boolean }>(`/rating/projects/${id}/level`, { minLevel }),
  deleteProject: (id: string) => apiClient.del<{ deleted: boolean }>(`/rating/projects/${id}`),
  createTask: (projectId: string, title: string) =>
    apiClient.post<{ id: string }>(`/rating/projects/${projectId}/tasks`, { title }),
  assign: (taskId: string) => apiClient.post<{ assigned: boolean }>(`/rating/tasks/${taskId}/assign`),
  setProgress: (taskId: string, progress: number) =>
    apiClient.patch<{ ok: boolean }>(`/rating/tasks/${taskId}/progress`, { progress }),
  returnTask: (taskId: string) => apiClient.post<{ returned: boolean }>(`/rating/tasks/${taskId}/return`),
  deleteTask: (taskId: string) => apiClient.del<{ deleted: boolean }>(`/rating/tasks/${taskId}`),
  addFinance: (projectId: string, incomeKopecks: number, expenseKopecks: number, note?: string) =>
    apiClient.post<{ ok: boolean }>(`/rating/projects/${projectId}/finance`, { incomeKopecks, expenseKopecks, note }),
};

export const notesApi = {
  list: () => apiClient.get<{ notes: Note[] }>("/notes"),
  createText: (text: string) => apiClient.post<{ note: Note }>("/notes", { text }),
  createFile: (file: File, text?: string) =>
    apiClient.uploadFile<{ note: Note }>(
      "/notes/file",
      file,
      text ? { "X-Note-Text": encodeURIComponent(text) } : undefined,
    ),
  del: (id: string) => apiClient.del<{ deleted: boolean }>(`/notes/${id}`),
};

export const pollsApi = {
  list: () => apiClient.get<{ polls: Poll[] }>("/polls"),
  create: (question: string, options: string[]) =>
    apiClient.post<{ id: string }>("/polls", { question, options }),
  vote: (optionId: string) => apiClient.post<{ voted: boolean }>(`/polls/options/${optionId}/vote`),
  close: (id: string) => apiClient.post<{ closed: boolean }>(`/polls/${id}/close`),
  del: (id: string) => apiClient.del<{ deleted: boolean }>(`/polls/${id}`),
};

export const callsApi = {
  iceConfig: () => apiClient.get<IceConfig>("/calls/ice-config"),
  history: (limit = 50, offset = 0) =>
    apiClient.get<{ calls: CallLogItem[] }>(`/calls/history?limit=${limit}&offset=${offset}`),
};

export const conditionsApi = {
  list: () => apiClient.get<{ conditions: LevelCondition[] }>("/conditions"),
  next: () => apiClient.get<{ condition: LevelCondition | null }>("/conditions/next"),
  set: (level: number, body: string) => apiClient.put<{ ok: boolean }>(`/conditions/${level}`, { body }),
};

export const adminApi = {
  users: (limit = 100, offset = 0) =>
    apiClient.get<{ users: User[] }>(`/admin/users?limit=${limit}&offset=${offset}`),
  changeRole: (userId: string, roleLevel: number) =>
    apiClient.patch<{ ok: boolean }>(`/admin/users/${userId}/role`, { roleLevel }),
  resetPassword: (userId: string, newPassword: string) =>
    apiClient.post<{ ok: boolean }>(`/admin/users/${userId}/reset-password`, { newPassword }),
  activate: (userId: string) => apiClient.post<{ ok: boolean }>(`/admin/users/${userId}/activate`),
  deactivate: (userId: string) => apiClient.post<{ ok: boolean }>(`/admin/users/${userId}/deactivate`),
  audit: (action = "", limit = 100, offset = 0) => {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
    if (action) params.set("action", action);
    return apiClient.get<{ entries: AuditEntry[] }>(`/admin/audit?${params.toString()}`);
  },
};
