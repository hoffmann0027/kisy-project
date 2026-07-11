import { useInfiniteQuery } from "@tanstack/react-query";
import { chatMediaApi } from "@shared/api/endpoints";
import type { ChatLinkPage, ChatMediaPage, ChatType } from "@shared/api/types";

// Context-panel tabs load lazily: enabled only while the panel is open and
// the tab is active, so opening a chat costs nothing extra.

export function useChatMedia(chatType: ChatType, chatId: string, kind: "media" | "files", enabled: boolean) {
  return useInfiniteQuery({
    queryKey: ["chat-media", chatType, chatId, kind],
    enabled,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => chatMediaApi.list(chatType, chatId, kind, pageParam),
    getNextPageParam: (last: ChatMediaPage) => (last.hasMore ? last.nextCursor : undefined),
  });
}

export function useChatLinks(chatType: ChatType, chatId: string, enabled: boolean) {
  return useInfiniteQuery({
    queryKey: ["chat-media", chatType, chatId, "links"],
    enabled,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => chatMediaApi.links(chatType, chatId, pageParam),
    getNextPageParam: (last: ChatLinkPage) => (last.hasMore ? last.nextCursor : undefined),
  });
}
