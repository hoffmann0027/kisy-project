import { Button, Spinner } from "@shared/ui";
import { t } from "@shared/i18n";
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
        <div className="feed__empty">{t("work.feed.membersOnly")}</div>
      ) : wall.isPending ? (
        <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
          <Spinner size={28} />
        </div>
      ) : posts.length === 0 ? (
        <div className="feed__empty">{canPost ? t("work.feed.wallEmptyCanPost") : t("work.feed.wallEmpty")}</div>
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
              {t("work.feed.showMore")}
            </Button>
          )}
        </>
      )}
    </div>
  );
}
