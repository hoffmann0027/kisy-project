import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { conditionsApi } from "@shared/api/endpoints";
import type { LevelCondition } from "@shared/api/types";

export const conditionKeys = {
  list: ["conditions", "all"] as const,
  next: ["conditions", "next"] as const,
};

// useAllConditions loads the full ladder (CEO only). Enabled only when the CEO
// panel is open.
export function useAllConditions(enabled: boolean) {
  return useQuery<LevelCondition[]>({
    queryKey: conditionKeys.list,
    enabled,
    queryFn: async () => (await conditionsApi.list()).conditions,
  });
}

// useNextCondition loads only the rule for the current user's next level.
export function useNextCondition(enabled: boolean) {
  return useQuery<LevelCondition | null>({
    queryKey: conditionKeys.next,
    enabled,
    queryFn: async () => (await conditionsApi.next()).condition,
  });
}

export function useSetCondition() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (a: { level: number; body: string }) => conditionsApi.set(a.level, a.body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: conditionKeys.list });
      qc.invalidateQueries({ queryKey: conditionKeys.next });
    },
  });
}
