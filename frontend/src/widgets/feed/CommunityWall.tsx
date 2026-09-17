import { Button, Spinner } from "@shared/ui";
import type { Group } from "@shared/api/types";
import { useCommunityPosts } from "@entities/post/queries";
import { PostComposer } from "@features/post-compose/PostComposer";
import { PostCard } from "./PostCard";
import "./feed.css";

// A community's own wall. Same cards as the feed; each still speaks as the
// community, but its header does not link back to the page it is already on.

// A closed community is members-only: the server refuses its wall to anyone
// who has not joined (audit A-01), so the screen says so instead of asking.
export function CommunityWall({
  group,
  canPost,
  membersOnly = false,
}: {
  group: Group;
  canPost: boolean;
  membersOnly?: boolean;
}) {
  const wall = useCommunityPosts(membersOnly ? null : group.id);
  const posts = wall.data?.pages.flatMap((p) => p.posts) ?? [];

  return (
    <div className="feed__scroll">
      {canPost && <PostComposer communityId={group.id} />}

      {membersOnly ? (
        <div className="feed__empty">
          Сообщество закрытое: записи видят только участники. Подайте заявку, чтобы вступить.
        </div>
      ) : wall.isPending ? (
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
