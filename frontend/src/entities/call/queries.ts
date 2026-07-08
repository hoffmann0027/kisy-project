import { useQuery } from "@tanstack/react-query";
import { callsApi } from "@shared/api/endpoints";
import type { CallLogItem } from "@shared/api/types";

export const callKeys = { history: ["calls", "history"] as const };

// useCallHistory loads the current user's call journal (newest first). Fetched
// only while the history panel is open.
export function useCallHistory(enabled: boolean) {
  return useQuery<CallLogItem[]>({
    queryKey: callKeys.history,
    enabled,
    queryFn: async () => (await callsApi.history()).calls,
  });
}
