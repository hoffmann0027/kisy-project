import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ratingApi } from "@shared/api/endpoints";
import type { RatingAnalytics, RatingBoard } from "@shared/api/types";

export const ratingKeys = {
  board: ["rating", "board"] as const,
  analytics: ["rating", "analytics"] as const,
};

export function useRatingBoard() {
  return useQuery<RatingBoard>({ queryKey: ratingKeys.board, queryFn: ratingApi.board });
}

export function useRatingAnalytics() {
  return useQuery<RatingAnalytics>({ queryKey: ratingKeys.analytics, queryFn: ratingApi.analytics });
}

// useRatingMutations bundles every board mutation; each invalidates the board
// and analytics so the columns and charts stay in sync after a change.
export function useRatingMutations() {
  const qc = useQueryClient();
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ratingKeys.board });
    qc.invalidateQueries({ queryKey: ratingKeys.analytics });
  };

  return {
    createProject: useMutation({
      mutationFn: (a: { title: string; difficulty: string; description?: string }) =>
        ratingApi.createProject(a.title, a.difficulty, a.description),
      onSuccess: refresh,
    }),
    deleteProject: useMutation({ mutationFn: (id: string) => ratingApi.deleteProject(id), onSuccess: refresh }),
    createTask: useMutation({
      mutationFn: (a: { projectId: string; title: string }) => ratingApi.createTask(a.projectId, a.title),
      onSuccess: refresh,
    }),
    assign: useMutation({ mutationFn: (taskId: string) => ratingApi.assign(taskId), onSuccess: refresh }),
    setProgress: useMutation({
      mutationFn: (a: { taskId: string; progress: number }) => ratingApi.setProgress(a.taskId, a.progress),
      onSuccess: refresh,
    }),
    addFinance: useMutation({
      mutationFn: (a: { taskId: string; incomeKopecks: number; expenseKopecks: number; note?: string }) =>
        ratingApi.addFinance(a.taskId, a.incomeKopecks, a.expenseKopecks, a.note),
      onSuccess: refresh,
    }),
  };
}
