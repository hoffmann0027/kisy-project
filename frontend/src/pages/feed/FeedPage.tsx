import { useState } from "react";
import { Button, Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { FeedSort } from "@shared/api/types";
import { useFeed } from "@entities/post/queries";
import { PostCard } from "@widgets/feed/PostCard";
import "@widgets/feed/feed.css";

// The shared feed: posts from every open community, ranked or chronological.
//
// Both orderings are offered because they answer different questions. "Новые"
// is what just happened; "Популярные" is what people reacted to, which on a
// quiet day is the only one with anything in it (the ranking and the reasoning
// behind it are in docs/spec/07-business-logic.md).

export function FeedPage() {
  const [sort, setSort] = useState<FeedSort>("popular");
  const feed = useFeed(sort);

  const posts = feed.data?.pages.flatMap((p) => p.posts) ?? [];

  return (
    <div className="feed">
      <header className="feed__head">
        <h1 className="feed__title">Лента</h1>
        <div className="feed__sort" role="tablist" aria-label="Сортировка ленты">
          {(["popular", "new"] as FeedSort[]).map((value) => (
            <button
              key={value}
              type="button"
              role="tab"
              aria-selected={sort === value}
              className={`feed__sort-btn${sort === value ? " feed__sort-btn--active" : ""}`}
              onClick={() => setSort(value)}
            >
              {value === "popular" ? "Популярные" : "Новые"}
            </button>
          ))}
        </div>
      </header>

      <div className="feed__scroll">
        {feed.isPending ? (
          <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
            <Spinner size={28} />
          </div>
        ) : posts.length === 0 ? (
          <div className="feed__empty">
            <div style={{ marginBottom: 10, color: "var(--color-text-tertiary)" }}>
              <Icon.Board size={36} />
            </div>
            Здесь появятся посты открытых сообществ. Вступите в сообщество или создайте своё — и лента
            оживёт.
          </div>
        ) : (
          <>
            {posts.map((post) => (
              <PostCard key={post.id} post={post} />
            ))}
            {feed.hasNextPage && (
              <Button
                variant="secondary"
                onClick={() => void feed.fetchNextPage()}
                loading={feed.isFetchingNextPage}
              >
                Показать ещё
              </Button>
            )}
          </>
        )}
      </div>
    </div>
  );
}
