import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { TabBar } from "./TabBar";

// The third tab is the one that differs between the two kinds of account: an
// account outside the role hierarchy has no rating board, so the feed takes
// that place rather than leaving a gap or a tab that answers 403.

function signIn(over: Partial<User>) {
  useAuthStore.setState({
    status: "authenticated",
    user: {
      id: "u1",
      username: "hamza",
      displayName: "Hamza",
      roleLevel: 1,
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

function renderBar() {
  return render(
    <MemoryRouter initialEntries={["/"]}>
      <TabBar onProfile={vi.fn()} />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  useAuthStore.setState({ status: "loading", user: null });
});

describe("the tab bar", () => {
  it("shows the rating board to an account inside the hierarchy", () => {
    signIn({ roleLevel: 3, accountKind: "invited" });
    renderBar();
    expect(screen.getByLabelText("Рейтинг")).toBeTruthy();
    expect(screen.queryByLabelText("Лента")).toBeNull();
  });

  it("puts the feed in its place for an account outside the hierarchy", () => {
    signIn({ roleLevel: null, accountKind: "basic" });
    renderBar();
    expect(screen.getByLabelText("Лента")).toBeTruthy();
    expect(screen.queryByLabelText("Рейтинг")).toBeNull();
  });

  it("does not offer the rating board to someone who is not in a clan", () => {
    // Level 10 is inside the hierarchy but off the board; the feed fills the
    // slot rather than a button that bounces back.
    signIn({ roleLevel: 10, accountKind: "invited" });
    renderBar();
    expect(screen.getByLabelText("Лента")).toBeTruthy();
    expect(screen.queryByLabelText("Рейтинг")).toBeNull();
  });

  it("keeps the tabs everyone has", () => {
    signIn({ roleLevel: null, accountKind: "basic" });
    renderBar();
    for (const label of ["Сообщения", "Сообщества", "Хаб", "Профиль"]) {
      expect(screen.getByLabelText(label)).toBeTruthy();
    }
  });
});
