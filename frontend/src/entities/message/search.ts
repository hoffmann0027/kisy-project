import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { searchApi } from "@shared/api/endpoints";

// useMessageSearch runs a full-text message search once the query is at least
// two characters, keeping previous results visible while the next page loads.
export function useMessageSearch(query: string) {
  const q = query.trim();
  return useQuery({
    queryKey: ["search", q],
    enabled: q.length >= 2,
    queryFn: async () => (await searchApi.messages(q)).results,
    placeholderData: keepPreviousData,
  });
}
