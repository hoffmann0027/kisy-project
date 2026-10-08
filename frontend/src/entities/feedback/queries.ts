import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { feedbackApi } from "@shared/api/endpoints";
import type { FeedbackItem, FeedbackPage, FeedbackScope } from "@shared/api/types";

export const feedbackKeys = {
  all: ["feedback"] as const,
  list: (scope: FeedbackScope) => ["feedback", scope] as const,
};

/** Levels that see everyone's unanswered feedback and answer it; 1 is the CEO. */
export const FEEDBACK_STAFF_MAX_LEVEL = 3;

export function canAnswerFeedback(roleLevel: number | null | undefined): boolean {
  return roleLevel != null && roleLevel >= 1 && roleLevel <= FEEDBACK_STAFF_MAX_LEVEL;
}

/** One entry per author in any 24 hours (the server holds the rule). */
export const FEEDBACK_WINDOW_MS = 24 * 60 * 60 * 1000;

/**
 * How long until the author may write again, from their newest entry; 0 when
 * they may write now.
 */
export function feedbackWait(newestCreatedAt: string | undefined, now = Date.now()): number {
  if (!newestCreatedAt) return 0;
  return Math.max(0, new Date(newestCreatedAt).getTime() + FEEDBACK_WINDOW_MS - now);
}

// useFeedback loads one scope newest-first with cursor paging.
export function useFeedback(scope: FeedbackScope, enabled: boolean) {
  return useInfiniteQuery({
    queryKey: feedbackKeys.list(scope),
    enabled,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => feedbackApi.list(scope, pageParam),
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
    onSuccess: () => qc.invalidateQueries({ queryKey: feedbackKeys.all }),
  });
}

export function useReplyFeedback() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) => feedbackApi.reply(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: feedbackKeys.all }),
  });
}

export function useDeleteFeedback() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => feedbackApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: feedbackKeys.all }),
  });
}
