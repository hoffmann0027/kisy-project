import { useQuery } from "@tanstack/react-query";
import { attachmentsApi } from "@shared/api/endpoints";

// The actor's upload policy (clearance-differentiated, served by the
// backend). Cached for the session — it changes only with the role.
export function useUploadLimit() {
  return useQuery({
    queryKey: ["upload-limit"],
    queryFn: () => attachmentsApi.limit(),
    staleTime: Infinity,
  });
}
