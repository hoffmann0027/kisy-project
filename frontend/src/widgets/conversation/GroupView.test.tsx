import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";
import type { Group, GroupViewer, User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { groupKeys } from "@entities/group/queries";
import { postKeys } from "@entities/post/queries";
import { GroupView } from "./GroupView";

// A community is run by its editors and read by everyone else. The board and
// the calendar are the editors' back office, people join a community only by
// their own choice, and an account outside the hierarchy is not shown a level
// field it can neither change nor learn anything from.

const ME = "u-owner";

function signIn(over: Partial<User>) {
  useAuthStore.setState({
    status: "authenticated",
    user: {
      id: ME,
      username: "owner",
      displayName: "Owner",
      roleLevel: 5,
      accountKind: "invited",
      avatarUrl: null,
      status: "online",
      isActive: true,
      lastSeen: null,
      createdAt: new Date().toISOString(),
      ...over,
    } as User,
  });
}

function community(over: Partial<Group> = {}): Group {
  return {
    id: "c1",
    name: "Горы",
    description: null,
    avatarUrl: null,
    minRoleLevel: null,
    kind: "community",
    isPublic: true,
    joinPolicy: "open",
    postPolicy: "editors",
    createdBy: ME,
    createdAt: new Date().toISOString(),
    ...over,
  };
}

function renderView(group: Group, viewer: GroupViewer) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  // Everything the screen asks for is already in the cache: these tests are
  // about what is offered, not about fetching.
  qc.setQueryData(groupKeys.viewer(group.id), viewer);
  qc.setQueryData(groupKeys.members(group.id), []);
  qc.setQueryData(postKeys.community(group.id), { pages: [{ posts: [], nextCursor: "" }], pageParams: [""] });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <GroupView group={group} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const reader: GroupViewer = { member: true, role: "member", canPost: false, canUseWorkspace: false };
const owner: GroupViewer = { member: true, role: "owner", canPost: true, canUseWorkspace: true };

beforeEach(() => {
  useAuthStore.setState({ status: "loading", user: null });
});

describe("a community's screen", () => {
  it("does not offer a plain reader the board or the calendar", () => {
    signIn({ id: "u-reader" });
    renderView(community(), reader);
    expect(screen.queryByText(/Доска/)).toBeNull();
    expect(screen.queryByText(/Календарь/)).toBeNull();
  });

  it("offers them to its editors", () => {
    signIn({});
    renderView(community(), owner);
    expect(screen.getByText(/Доска/)).toBeTruthy();
    expect(screen.getByText(/Календарь/)).toBeTruthy();
  });

  it("never offers to add people, even to its founder", () => {
    signIn({});
    renderView(community(), owner);
    fireEvent.click(screen.getByTitle("Участники"));
    expect(screen.queryByText("Добавить участника")).toBeNull();
  });

  it("hides the level field from an account outside the hierarchy", () => {
    signIn({ roleLevel: null, accountKind: "basic" });
    renderView(community(), owner);
    fireEvent.click(screen.getByTitle("Участники"));
    expect(screen.getByText("Удалить сообщество")).toBeTruthy(); // the modal is open
    expect(screen.queryByText("Уровень доступа")).toBeNull();
    expect(screen.queryByText("Без ограничения по уровню")).toBeNull();
  });

  it("still shows it to an invited account", () => {
    signIn({});
    renderView(community(), owner);
    fireEvent.click(screen.getByTitle("Участники"));
    expect(screen.getByText("Уровень доступа")).toBeTruthy();
  });
});
