import { useInfiniteQuery, useMutation, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { feedApi, postsApi } from "@shared/api/endpoints";
import type { FeedSort, PostPage, PostReaction } from "@shared/api/types";

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

/**
 * A post's reactions after the viewer chooses `emoji` (on) or takes it back
 * (off). One reaction per reader: whatever they had before is released first,
 * which is exactly what the server does (migration 45).
 */
export function applyReaction(reactions: PostReaction[], emoji: string, on: boolean): PostReaction[] {
  const released = reactions
    .map((r) => (r.mine ? { ...r, count: r.count - 1, mine: false } : r))
    .filter((r) => r.count > 0);
  if (!on) return released;
  const existing = released.find((r) => r.emoji === emoji);
  if (existing) return released.map((r) => (r === existing ? { ...r, count: r.count + 1, mine: true } : r));
  return [...released, { emoji, count: 1, mine: true }];
}

export function useReactToPost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { postId: string; communityId: string; emoji: string; on: boolean }) =>
      args.on ? postsApi.react(args.postId, args.emoji) : postsApi.unreact(args.postId, args.emoji),
    // Shown at once rather than after a round trip: with one reaction per
    // reader, a tap that lights the new chip while the old one stays lit for a
    // moment reads as "I now have two".
    onMutate: async (args) => {
      // A refetch already on its way would land on top of the optimistic
      // state with the old counts.
      await Promise.all([
        qc.cancelQueries({ queryKey: ["feed"] }),
        qc.cancelQueries({ queryKey: postKeys.community(args.communityId) }),
      ]);
      const update = (data: InfiniteData<PostPage> | undefined) =>
        data && {
          ...data,
          pages: data.pages.map((page) => ({
            ...page,
            posts: page.posts.map((p) =>
              p.id === args.postId ? { ...p, reactions: applyReaction(p.reactions, args.emoji, args.on) } : p,
            ),
          })),
        };
      qc.setQueriesData<InfiniteData<PostPage>>({ queryKey: ["feed"] }, update);
      qc.setQueriesData<InfiniteData<PostPage>>({ queryKey: postKeys.community(args.communityId) }, update);
    },
    // Success or failure, the server's count is the one to keep.
    onSettled: (_res, _err, args) => invalidatePosts(qc, args.communityId),
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
