import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { feedApi, postsApi } from "@shared/api/endpoints";
import type { FeedSort, PostPage } from "@shared/api/types";

export const postKeys = {
  feed: (sort: FeedSort) => ["feed", sort] as const,
  community: (communityId: string) => ["community-posts", communityId] as const,
};

/**
 * The shared feed. Paged rather than loaded whole: "popular" is a ranking the
 * server recomputes on a timer, and asking for all of it would mean asking for
 * every post in the last month.
 */
export function useFeed(sort: FeedSort) {
  return useInfiniteQuery({
    queryKey: postKeys.feed(sort),
    initialPageParam: "",
    queryFn: ({ pageParam }) => feedApi.list(sort, pageParam as string),
    getNextPageParam: (last: PostPage) => last.nextCursor || undefined,
  });
}

/** One community's wall. */
export function useCommunityPosts(communityId: string | null) {
  return useInfiniteQuery({
    queryKey: postKeys.community(communityId ?? ""),
    enabled: !!communityId,
    initialPageParam: "",
    queryFn: ({ pageParam }) => postsApi.ofCommunity(communityId as string, pageParam as string),
    getNextPageParam: (last: PostPage) => last.nextCursor || undefined,
  });
}

/** Everything a post touches: its community's wall and both feed orderings. */
function invalidatePosts(qc: ReturnType<typeof useQueryClient>, communityId?: string) {
  if (communityId) qc.invalidateQueries({ queryKey: postKeys.community(communityId) });
  qc.invalidateQueries({ queryKey: ["feed"] });
}

export function useCreatePost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (args: { communityId: string; text: string; files: File[] }) => {
      const { post } = await postsApi.create(args.communityId, args.text);
      // Files travel after the post because a file and a JSON body do not
      // share a request. Uploading them in order keeps the gallery in the
      // order the author picked.
      let latest = post;
      for (const file of args.files) {
        latest = (await postsApi.attach(post.id, file)).post;
      }
      return latest;
    },
    onSuccess: (_post, args) => invalidatePosts(qc, args.communityId),
  });
}

export function useDeletePost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { postId: string; communityId: string }) => postsApi.remove(args.postId),
    onSuccess: (_res, args) => invalidatePosts(qc, args.communityId),
  });
}

export function useReactToPost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { postId: string; communityId: string; emoji: string; on: boolean }) =>
      args.on ? postsApi.react(args.postId, args.emoji) : postsApi.unreact(args.postId, args.emoji),
    onSuccess: (_res, args) => invalidatePosts(qc, args.communityId),
  });
}

/** Drops a community out of this reader's feed. Membership is untouched. */
export function useHideCommunityFromFeed() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { communityId: string; hidden: boolean }) =>
      args.hidden ? feedApi.hide(args.communityId) : feedApi.show(args.communityId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["feed"] }),
  });
}
