import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Group, GroupMember, GroupRole, User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { groupKeys } from "@entities/group/queries";
import { GroupMembersModal } from "./GroupMembersModal";

// A community could be joined but never left, and nobody who ran it could
// show anyone the door. The keys follow the server's rule: a member leaves
// (the founder deletes instead); the founder, editors and moderators remove
// and ban those ranked below them, and the CEO anyone but the founder.

const FOUNDER = "u-founder";

function user(id: string, over: Partial<User> = {}): User {
  return {
    id,
    username: id,
    displayName: id,
    roleLevel: 8,
    accountKind: "invited",
    avatarUrl: null,
    status: "online",
    isActive: true,
    lastSeen: null,
    createdAt: new Date().toISOString(),
    ...over,
  } as User;
}

function signIn(id: string, over: Partial<User> = {}) {
  useAuthStore.setState({ status: "authenticated", user: user(id, over) });
}

const community: Group = {
  id: "c1",
  name: "Vibe",
  description: null,
  avatarUrl: null,
  minRoleLevel: null,
  kind: "community",
  isPublic: true,
  joinPolicy: "open",
  postPolicy: "editors",
  createdBy: FOUNDER,
  createdAt: new Date().toISOString(),
};

const MEMBERS: GroupMember[] = (
  [
    [FOUNDER, "owner"],
    ["u-editor", "editor"],
    ["u-moderator", "moderator"],
    ["u-reader", "member"],
    ["u-other", "member"],
  ] as [string, GroupRole][]
).map(([id, role]) => ({ user: user(id), role }));

function renderModal() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  qc.setQueryData(groupKeys.members(community.id), MEMBERS);
  qc.setQueryData(groupKeys.requests(community.id), []);
  qc.setQueryData(groupKeys.bans(community.id), []);
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <GroupMembersModal group={community} canAdd={false} open onClose={vi.fn()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const removeKeys = () => screen.queryAllByLabelText(/^Исключить: /).map((b) => b.getAttribute("aria-label")!.replace("Исключить: ", ""));
const banKeys = () => screen.queryAllByLabelText(/^Заблокировать: /).map((b) => b.getAttribute("aria-label")!.replace("Заблокировать: ", ""));

beforeEach(() => {
  useAuthStore.setState({ status: "loading", user: null });
});

describe("leaving a community", () => {
  it("offers a member the way out", () => {
    signIn("u-reader");
    renderModal();
    expect(screen.getByText("Покинуть сообщество")).toBeTruthy();
    expect(screen.queryByText("Удалить сообщество")).toBeNull();
  });

  it("offers the founder deletion, never leaving", () => {
    signIn(FOUNDER);
    renderModal();
    expect(screen.queryByText("Покинуть сообщество")).toBeNull();
    expect(screen.getByText("Удалить сообщество")).toBeTruthy();
  });
});

describe("removing and banning", () => {
  it("gives a plain member no keys at all", () => {
    signIn("u-reader");
    renderModal();
    expect(removeKeys()).toEqual([]);
    expect(banKeys()).toEqual([]);
  });

  it("lets a moderator act on members only", () => {
    signIn("u-moderator");
    renderModal();
    expect(removeKeys()).toEqual(["u-reader", "u-other"]);
    expect(banKeys()).toEqual(["u-reader", "u-other"]);
  });

  it("lets an editor act on moderators and members, never on the founder or themselves", () => {
    signIn("u-editor");
    renderModal();
    expect(removeKeys()).toEqual(["u-moderator", "u-reader", "u-other"]);
  });

  it("lets the founder act on everyone else", () => {
    signIn(FOUNDER);
    renderModal();
    expect(removeKeys()).toEqual(["u-editor", "u-moderator", "u-reader", "u-other"]);
  });

  it("lets the CEO act on everyone but the founder, from outside the community", () => {
    signIn("u-ceo", { roleLevel: 1 });
    renderModal();
    expect(removeKeys()).toEqual(["u-editor", "u-moderator", "u-reader", "u-other"]);
    expect(banKeys()).toEqual(["u-editor", "u-moderator", "u-reader", "u-other"]);
  });
});
