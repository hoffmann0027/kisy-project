import { useNavigate } from "react-router-dom";
import { Avatar, Button, IconButton, toast } from "@shared/ui";
import { ApiImage } from "@shared/ui/ApiImage";
import { Icon } from "@shared/ui/icons";
import { formatRelative } from "@shared/lib/format";
import type { Post } from "@shared/api/types";
import { useJoinGroup } from "@entities/group/queries";
import { useDeletePost, useReactToPost } from "@entities/post/queries";

// One post in the feed or on a community's wall.
//
// The community strip along the top is not decoration. A post in the shared
// feed comes from somewhere the reader may not belong to, and a post whose
// origin cannot be opened or joined is a dead end — you see something
// interesting and have nowhere to go.

/** The reactions offered on a post. Same set as a chat message's quick row. */
const QUICK_REACTIONS = ["👍", "🔥", "❤️", "😂", "👏"];

interface Props {
  post: Post;
  /** On a community's own wall the origin strip would repeat the page title. */
  showCommunity?: boolean;
}

export function PostCard({ post, showCommunity = true }: Props) {
  const navigate = useNavigate();
  const react = useReactToPost();
  const remove = useDeletePost();
  const join = useJoinGroup();

  const toggle = (emoji: string) => {
    const mine = post.reactions.some((r) => r.emoji === emoji && r.mine);
    react.mutate(
      { postId: post.id, communityId: post.community.id, emoji, on: !mine },
      { onError: () => toast.error("Не удалось поставить реакцию") },
    );
  };

  const joinCommunity = () => {
    join.mutate(post.community.id, {
      onSuccess: () =>
        toast.success(
          post.community.joinPolicy === "open"
            ? "Вы вступили в сообщество"
            : "Заявка отправлена — её рассмотрят администраторы",
        ),
      onError: () => toast.error("Не удалось вступить"),
    });
  };

  return (
    <article className="post">
      {showCommunity && (
        <header className="post__origin">
          <button
            type="button"
            className="post__origin-link"
            onClick={() => navigate(`/group/${post.community.id}`)}
          >
            <Avatar name={post.community.name} url={post.community.avatarUrl} size={28} />
            <span className="post__origin-name">{post.community.name}</span>
          </button>
          {!post.community.isMember && (
            <Button variant="secondary" onClick={joinCommunity} loading={join.isPending}>
              {post.community.joinPolicy === "open" ? "Вступить" : "Подать заявку"}
            </Button>
          )}
        </header>
      )}

      <div className="post__head">
        <Avatar name={post.author.displayName} url={post.author.avatarUrl} size={36} />
        <div className="post__who">
          <div className="post__author">{post.author.displayName}</div>
          <div className="post__time">
            {formatRelative(post.createdAt)}
            {post.editedAt && " · изменено"}
          </div>
        </div>
        {post.canDelete && (
          <IconButton
            label="Удалить пост"
            onClick={() =>
              remove.mutate(
                { postId: post.id, communityId: post.community.id },
                { onError: () => toast.error("Не удалось удалить пост") },
              )
            }
          >
            <Icon.Trash size={18} />
          </IconButton>
        )}
      </div>

      {post.text && <p className="post__text">{post.text}</p>}

      {post.media.length > 0 && (
        <div className="post__media">
          {post.media.map((m) =>
            m.kind === "image" ? (
              <ApiImage key={m.id} src={m.url} alt={m.fileName} className="post__image" />
            ) : m.kind === "audio" ? (
              <audio key={m.id} className="post__audio" controls src={m.url} />
            ) : m.kind === "video" ? (
              <video key={m.id} className="post__video" controls src={m.url} />
            ) : (
              <a key={m.id} className="post__file" href={m.url} target="_blank" rel="noreferrer">
                <Icon.Paperclip size={16} />
                {m.fileName}
              </a>
            ),
          )}
        </div>
      )}

      <footer className="post__reactions">
        {QUICK_REACTIONS.map((emoji) => {
          const summary = post.reactions.find((r) => r.emoji === emoji);
          return (
            <button
              key={emoji}
              type="button"
              className={`post__reaction${summary?.mine ? " post__reaction--mine" : ""}`}
              onClick={() => toggle(emoji)}
            >
              <span>{emoji}</span>
              {summary && summary.count > 0 && <span className="post__reaction-count">{summary.count}</span>}
            </button>
          );
        })}
        {/* Reactions with an emoji outside the quick row still show their count. */}
        {post.reactions
          .filter((r) => !QUICK_REACTIONS.includes(r.emoji))
          .map((r) => (
            <button
              key={r.emoji}
              type="button"
              className={`post__reaction${r.mine ? " post__reaction--mine" : ""}`}
              onClick={() => toggle(r.emoji)}
            >
              <span>{r.emoji}</span>
              <span className="post__reaction-count">{r.count}</span>
            </button>
          ))}
      </footer>
    </article>
  );
}
