import { apiClient } from "./client";
import type {
  AuditEntry,
  Board,
  CardInput,
  Chat,
  ChatType,
  Favorite,
  Group,
  Invitation,
  Message,
  MessagePage,
  Notification,
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
  directory: (search = "", limit = 25) => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (search) params.set("search", search);
    return apiClient.get<{ users: User[] }>(`/users/directory?${params.toString()}`);
  },
};

export const invitesApi = {
  create: () => apiClient.post<Invitation>("/invites"),
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
  send: (chatType: ChatType, chatId: string, text: string, replyTo?: string) =>
    apiClient.post<{ message: Message }>("/messages", { chatType, chatId, text, replyTo }),
  remove: (messageId: string) => apiClient.del<{ deleted: boolean }>(`/messages/${messageId}`),
  addReaction: (messageId: string, emoji: string) =>
    apiClient.post<{ ok: boolean }>(`/messages/${messageId}/reactions`, { emoji }),
  removeReaction: (messageId: string, emoji: string) =>
    apiClient.del<{ ok: boolean }>(`/messages/${messageId}/reactions`, { emoji }),
  markRead: (chatType: ChatType, chatId: string, messageId: string) =>
    apiClient.post<{ ok: boolean }>("/read", { chatType, chatId, messageId }),
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
