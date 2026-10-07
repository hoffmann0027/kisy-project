import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { Post } from "@shared/api/types";

// A post could be reported by the server but not from the app: the dialog knew
// "post", the API accepted it, and no button led there — so the automatic
// hiding after five reports could never trigger. These pin the way in.

vi.mock("@entities/group/queries", () => ({ useJoinGroup: () => ({ mutate: vi.fn(), isPending: false }) }));
vi.mock("@entities/post/queries", () => ({
  useReactToPost: () => ({ mutate: vi.fn() }),
  useDeletePost: () => ({ mutate: vi.fn() }),
}));
const create = vi.hoisted(() => vi.fn(async () => ({ id: "r1" })));
vi.mock("@shared/api/endpoints", () => ({ reportsApi: { create } }));

const { PostCard } = await import("./PostCard");

function post(canDelete: boolean): Post {
  return {
    id: "p1",
    text: "запись",
    createdAt: new Date().toISOString(),
    editedAt: null,
    community: {
      id: "c1",
      name: "Сообщество",
      avatarUrl: null,
      verified: false,
      isMember: true,
      joinPolicy: "open",
    },
    media: [],
    reactions: [],
    canDelete,
  };
}

describe("PostCard reporting", () => {
  it("lets a reader report someone else's post", async () => {
    render(
      <MemoryRouter>
        <PostCard post={post(false)} />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Пожаловаться на запись" }));
    fireEvent.click(screen.getByRole("button", { name: "Отправить" }));
    await vi.waitFor(() => expect(create).toHaveBeenCalledWith(expect.objectContaining({ targetKind: "post", targetId: "p1" })));
  });

  it("offers deletion, not a report, to whoever can delete the post", () => {
    // The author, the community's editors and the CEO remove a post; reporting
    // it to themselves would only put it in a queue they already run.
    render(
      <MemoryRouter>
        <PostCard post={post(true)} />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("button", { name: "Пожаловаться на запись" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Удалить пост" })).toBeInTheDocument();
  });
});
