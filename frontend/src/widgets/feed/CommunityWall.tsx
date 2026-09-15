import { Button, Spinner } from "@shared/ui";
import type { Group } from "@shared/api/types";
import { useCommunityPosts } from "@entities/post/queries";
import { PostComposer } from "@features/post-compose/PostComposer";
import { PostCard } from "./PostCard";
import "./feed.css";

// A community's own wall. Same cards as the feed, minus the origin strip —
// on the community's own page, saying where each post came from would repeat
// the page title once per post.

export function CommunityWall({ group, canPost }: { group: Group; canPost: boolean }) {
  const wall = useCommunityPosts(group.id);
  const posts = wall.data?.pages.flatMap((p) => p.posts) ?? [];

  return (
    <div className="feed__scroll">
      {canPost && <PostComposer communityId={group.id} />}

      {wall.isPending ? (
        <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
          <Spinner size={28} />
        </div>
      ) : posts.length === 0 ? (
        <div className="feed__empty">
          {canPost
            ? "Пока пусто. Напишите первый пост — его увидят участники сообщества."
            : "Пока здесь нет постов."}
        </div>
      ) : (
        <>
          {posts.map((post) => (
            <PostCard key={post.id} post={post} showCommunity={false} />
          ))}
          {wall.hasNextPage && (
            <Button
              variant="secondary"
              onClick={() => void wall.fetchNextPage()}
              loading={wall.isFetchingNextPage}
            >
              Показать ещё
            </Button>
          )}
        </>
      )}
    </div>
  );
}
