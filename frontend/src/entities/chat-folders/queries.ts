// Chat folders + archive (UPD3 stage H). Folders and the archive are
// personal metadata: filtering happens client-side against the
// access-filtered chat/group lists, so an inaccessible chat referenced by a
// folder simply never renders.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatFoldersApi, type ChatFolder } from "@shared/api/endpoints";
import type { ChatType } from "@shared/api/types";

const foldersKey = ["chat-folders"] as const;
const archivedKey = ["chat-archived"] as const;

export function chatKey(chatType: ChatType, chatId: string): string {
  return `${chatType}:${chatId}`;
}

export function useFolders() {
  const query = useQuery({
    queryKey: foldersKey,
    queryFn: async () => (await chatFoldersApi.list()).folders,
  });
  return { ...query, folders: query.data ?? [] };
}

/** Set of "chatType:chatId" for one folder — for quick membership checks. */
export function folderChatSet(folder: ChatFolder): Set<string> {
  return new Set(folder.items.map((i) => chatKey(i.chatType, i.chatId)));
}

export function useArchived() {
  const query = useQuery({
    queryKey: archivedKey,
    queryFn: async () => (await chatFoldersApi.listArchived()).archived,
  });
  const archivedSet = new Set((query.data ?? []).map((e) => chatKey(e.chatType, e.chatId)));
  return { ...query, archivedSet };
}

export function isArchived(set: Set<string>, chatType: ChatType, chatId: string): boolean {
  return set.has(chatKey(chatType, chatId));
}

function useInvalidate(key: readonly string[]) {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: key });
}

export function useCreateFolder() {
  const invalidate = useInvalidate(foldersKey);
  return useMutation({
    mutationFn: (name: string) => chatFoldersApi.create(name),
    onSuccess: invalidate,
  });
}

export function useRenameFolder() {
  const invalidate = useInvalidate(foldersKey);
  return useMutation({
    mutationFn: (args: { id: string; name: string }) => chatFoldersApi.rename(args.id, args.name),
    onSuccess: invalidate,
  });
}

export function useDeleteFolder() {
  const invalidate = useInvalidate(foldersKey);
  return useMutation({
    mutationFn: (id: string) => chatFoldersApi.remove(id),
    onSuccess: invalidate,
  });
}

export function useReorderFolders() {
  const invalidate = useInvalidate(foldersKey);
  return useMutation({
    mutationFn: (folderIds: string[]) => chatFoldersApi.reorder(folderIds),
    onSuccess: invalidate,
  });
}

export function useFolderItem() {
  const invalidate = useInvalidate(foldersKey);
  return useMutation({
    mutationFn: (args: { folderId: string; chatType: ChatType; chatId: string; add: boolean }) =>
      args.add
        ? chatFoldersApi.addItem(args.folderId, args.chatType, args.chatId)
        : chatFoldersApi.removeItem(args.folderId, args.chatType, args.chatId),
    onSuccess: invalidate,
  });
}

export function useArchiveChat() {
  const invalidate = useInvalidate(archivedKey);
  return useMutation({
    mutationFn: (args: { chatType: ChatType; chatId: string; archive: boolean }) =>
      args.archive
        ? chatFoldersApi.archive(args.chatType, args.chatId)
        : chatFoldersApi.unarchive(args.chatType, args.chatId),
    onSuccess: invalidate,
  });
}
