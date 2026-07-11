import { useQuery } from "@tanstack/react-query";
import { linkPreviewApi } from "@shared/api/endpoints";

// Fetch a URL's preview lazily. Enabled per bubble once the URL is known;
// the server caches results so repeated links are cheap. A failed preview is
// simply not rendered (no error surfaced in the timeline).
export function useLinkPreview(url: string | null, enabled: boolean) {
  return useQuery({
    queryKey: ["link-preview", url],
    enabled: enabled && !!url,
    retry: false,
    staleTime: 60 * 60 * 1000,
    queryFn: async () => {
      const { preview } = await linkPreviewApi.fetch(url as string);
      return preview;
    },
  });
}
