import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { boardsApi } from "@shared/api/endpoints";
import type { CardInput } from "@shared/api/types";
import { ApiError } from "@shared/api/envelope";

export const boardKeys = { board: (groupId: string) => ["board", groupId] as const };

// useBoard loads a group's task board. A 404 (no board yet) is not an
// error we retry — it just means the founder hasn't created one.
export function useBoard(groupId: string | null) {
  return useQuery({
    queryKey: groupId ? boardKeys.board(groupId) : ["board", "none"],
    enabled: !!groupId,
    retry: (count, err) => !(err instanceof ApiError && err.status === 404) && count < 2,
    queryFn: async () => (await boardsApi.get(groupId as string)).board,
  });
}

export function useBoardMutations(groupId: string) {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: boardKeys.board(groupId) });
  const opts = { onSuccess: invalidate };

  return {
    createBoard: useMutation({ mutationFn: (title: string) => boardsApi.create(groupId, title), ...opts }),
    addColumn: useMutation({
      mutationFn: (args: { boardId: string; title: string }) => boardsApi.addColumn(args.boardId, args.title),
      ...opts,
    }),
    renameColumn: useMutation({
      mutationFn: (args: { columnId: string; title: string }) => boardsApi.renameColumn(args.columnId, args.title),
      ...opts,
    }),
    deleteColumn: useMutation({ mutationFn: (columnId: string) => boardsApi.deleteColumn(columnId), ...opts }),
    createCard: useMutation({
      mutationFn: (args: { columnId: string; input: CardInput }) => boardsApi.createCard(args.columnId, args.input),
      ...opts,
    }),
    updateCard: useMutation({
      mutationFn: (args: { cardId: string; input: CardInput }) => boardsApi.updateCard(args.cardId, args.input),
      ...opts,
    }),
    moveCard: useMutation({
      mutationFn: (args: { cardId: string; columnId: string; index: number }) =>
        boardsApi.moveCard(args.cardId, args.columnId, args.index),
      ...opts,
    }),
    deleteCard: useMutation({ mutationFn: (cardId: string) => boardsApi.deleteCard(cardId), ...opts }),
  };
}
