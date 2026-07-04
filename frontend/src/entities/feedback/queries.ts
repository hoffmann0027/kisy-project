import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { feedbackApi } from "@shared/api/endpoints";
import type { FeedbackItem, FeedbackPage } from "@shared/api/types";

export const feedbackKeys = { list: ["feedback"] as const };

// useFeedback loads the feedback board newest-first with cursor paging.
export function useFeedback(enabled: boolean) {
  return useInfiniteQuery({
    queryKey: feedbackKeys.list,
    enabled,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => feedbackApi.list(pageParam),
    getNextPageParam: (last: FeedbackPage) => (last.hasMore ? (last.nextCursor ?? undefined) : undefined),
  });
}

export function flattenFeedback(pages: FeedbackPage[] | undefined): FeedbackItem[] {
  return pages?.flatMap((p) => p.items) ?? [];
}

export function useCreateFeedback() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => feedbackApi.create(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: feedbackKeys.list }),
  });
}

export function useDeleteFeedback() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => feedbackApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: feedbackKeys.list }),
  });
}
