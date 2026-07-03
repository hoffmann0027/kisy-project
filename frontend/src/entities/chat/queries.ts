import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatsApi, favoritesApi } from "@shared/api/endpoints";
import type { Chat, Favorite } from "@shared/api/types";

export const chatKeys = {
  list: ["chats"] as const,
  favorites: ["favorites"] as const,
};

export function useChats() {
  return useQuery({
    queryKey: chatKeys.list,
    queryFn: async () => (await chatsApi.list()).chats,
  });
}

export function useFavorites() {
  return useQuery({
    queryKey: chatKeys.favorites,
    queryFn: async () => (await favoritesApi.list()).favorites,
  });
}

export function useOpenChat() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (userId: string) => (await chatsApi.open(userId)).chat,
    onSuccess: (chat) => {
      qc.setQueryData<Chat[]>(chatKeys.list, (prev) => {
        if (!prev) return [chat];
        if (prev.some((c) => c.id === chat.id)) return prev;
        return [chat, ...prev];
      });
    },
  });
}

export function useToggleFavorite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (args: { fav: Favorite; remove?: boolean }) => {
      if (args.remove) return favoritesApi.remove(args.fav.chatType, args.fav.chatId);
      return favoritesApi.set(args.fav);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: chatKeys.favorites }),
  });
}
