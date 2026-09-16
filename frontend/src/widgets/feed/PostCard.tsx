import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Avatar, Button, EmojiPicker, IconButton, VerifiedName, toast } from "@shared/ui";
import { ApiImage } from "@shared/ui/ApiImage";
import { Icon } from "@shared/ui/icons";
import { formatRelative } from "@shared/lib/format";
import type { Post } from "@shared/api/types";
import { useJoinGroup } from "@entities/group/queries";
import { useDeletePost, useReactToPost } from "@entities/post/queries";

// One post in the feed or on a community's wall.
//
// A post speaks as its community — its avatar and name head the card, the way
// a channel's do — so the header is also the way out: in the shared feed it
// opens the community and, for a reader who is not in it, offers to join. A
// post whose origin cannot be opened or joined is a dead end.

/** The reactions offered on a post without opening the picker. */
const QUICK_REACTIONS = ["👍", "🔥", "❤️", "😂", "👏"];

interface Props {
  post: Post;
  /** In the shared feed the header links to the community and offers to join;
      on the community's own wall that would only point back at this page. */
  showCommunity?: boolean;
}

/** A touch screen raises its keyboard for a focused field — over the grid. */
function isTouchScreen() {
  return typeof window !== "undefined" && !!window.matchMedia?.("(pointer: coarse)").matches;
}

export function PostCard({ post, showCommunity = true }: Props) {
  const navigate = useNavigate();
  const react = useReactToPost();
  const remove = useDeletePost();
  const join = useJoinGroup();
  const [pickerOpen, setPickerOpen] = useState(false);
  const reactionsRef = useRef<HTMLElement>(null);

  // The picker opens below the reactions so it is never cut off by the top of
  // the list; this brings it on screen when the post sits low in the viewport.
  useEffect(() => {
    if (!pickerOpen) return;
    reactionsRef.current?.querySelector(".emojipick")?.scrollIntoView?.({ block: "nearest", behavior: "smooth" });
  }, [pickerOpen]);

  // One reaction per reader: tapping your own takes it back, tapping another
  // replaces it (the server keeps one row per person).
  const choose = (emoji: string) => {
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

  const origin = (
    <>
      <Avatar name={post.community.name} url={post.community.avatarUrl} size={40} />
      <span className="post__who">
        <VerifiedName className="post__community" name={post.community.name} verified={post.community.verified} subject="group" />
        <span className="post__time">
          {formatRelative(post.createdAt)}
          {post.editedAt && " · изменено"}
        </span>
      </span>
    </>
  );

  // Quick emoji first, then anything else someone picked from the full set.
  const extra = post.reactions.filter((r) => !QUICK_REACTIONS.includes(r.emoji));
  const chips = [
    ...QUICK_REACTIONS.map((emoji) => post.reactions.find((r) => r.emoji === emoji) ?? { emoji, count: 0, mine: false }),
    ...extra,
  ];

  return (
    <article className="post">
      <header className="post__head">
        {showCommunity ? (
          <button type="button" className="post__origin" onClick={() => navigate(`/group/${post.community.id}`)}>
            {origin}
          </button>
        ) : (
          <div className="post__origin">{origin}</div>
        )}
        {showCommunity && !post.community.isMember && (
          <Button variant="secondary" onClick={joinCommunity} loading={join.isPending}>
            {post.community.joinPolicy === "open" ? "Вступить" : "Подать заявку"}
          </Button>
        )}
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
      </header>

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

      <footer className="post__reactions" ref={reactionsRef}>
        {chips.map((r) => (
          <button
            key={r.emoji}
            type="button"
            className={`post__reaction${r.mine ? " post__reaction--mine" : ""}`}
            aria-pressed={r.mine}
            onClick={() => choose(r.emoji)}
          >
            <span>{r.emoji}</span>
            {r.count > 0 && <span className="post__reaction-count">{r.count}</span>}
          </button>
        ))}
        <button
          type="button"
          className="post__reaction post__reaction-more"
          aria-label="Другие эмодзи"
          aria-expanded={pickerOpen}
          data-picker-toggle={post.id}
          onClick={() => setPickerOpen((v) => !v)}
        >
          <Icon.Plus size={16} />
        </button>
        {pickerOpen && (
          <EmojiPicker
            // This post's own toggle only: the "+" on the next card is an
            // outside click, and it must close this picker as it opens its own.
            ignoreSelector={`[data-picker-toggle="${post.id}"]`}
            autoFocusSearch={!isTouchScreen()}
            onPick={(emoji) => {
              setPickerOpen(false);
              // From the picker it is always a choice, never a take-back: the
              // emoji you already have is right there on the card to tap.
              if (!post.reactions.some((r) => r.emoji === emoji && r.mine)) choose(emoji);
            }}
            onClose={() => setPickerOpen(false)}
          />
        )}
      </footer>
    </article>
  );
}
