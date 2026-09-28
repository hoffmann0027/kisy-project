import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { usersApi } from "@shared/api/endpoints";

// Who this account has blocked (E-02). Kept in one query so the chat header,
// the profile settings and the feed all read the same answer.

export const blockKeys = { list: ["blocks"] as const };

export function useBlocks() {
  const { data } = useQuery({
    queryKey: blockKeys.list,
    queryFn: () => usersApi.blocks(),
    staleTime: 60_000,
  });
  return data?.blocks ?? [];
}

export function useIsBlocked(userId: string | undefined): boolean {
  const blocks = useBlocks();
  return !!userId && blocks.some((b) => b.userId === userId);
}

export function useSetBlocked() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, blocked }: { userId: string; blocked: boolean }) =>
      blocked ? usersApi.block(userId) : usersApi.unblock(userId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: blockKeys.list });
      // A block changes what the feed, the chat list and search may show.
      void qc.invalidateQueries({ queryKey: ["feed"] });
      void qc.invalidateQueries({ queryKey: ["chats"] });
    },
  });
}
