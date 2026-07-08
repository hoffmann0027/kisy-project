import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { pollsApi } from "@shared/api/endpoints";
import type { Poll } from "@shared/api/types";

export const pollKeys = { list: ["polls"] as const };

// usePolls loads every poll with options, tallies and voters. Real-time
// poll.changed events invalidate this key so votes appear live.
export function usePolls(enabled: boolean) {
  return useQuery<Poll[]>({
    queryKey: pollKeys.list,
    enabled,
    queryFn: async () => (await pollsApi.list()).polls,
  });
}

export function usePollMutations() {
  const qc = useQueryClient();
  const refresh = () => qc.invalidateQueries({ queryKey: pollKeys.list });

  return {
    create: useMutation({
      mutationFn: (a: { question: string; options: string[] }) => pollsApi.create(a.question, a.options),
      onSuccess: refresh,
    }),
    vote: useMutation({ mutationFn: (optionId: string) => pollsApi.vote(optionId), onSuccess: refresh }),
    close: useMutation({ mutationFn: (id: string) => pollsApi.close(id), onSuccess: refresh }),
    del: useMutation({ mutationFn: (id: string) => pollsApi.del(id), onSuccess: refresh }),
  };
}
