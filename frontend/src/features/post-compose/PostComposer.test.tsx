import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// A brand-new account may not publish yet (backend: internal/quarantine). The
// composer says so instead of offering a form whose submit will be refused.

const store = vi.hoisted(() => ({ quarantine: null as unknown }));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (selector: (s: typeof store) => unknown) => selector(store),
}));
vi.mock("@entities/post/queries", () => ({
  useCreatePost: () => ({ mutate: vi.fn(), isPending: false }),
}));

const { PostComposer } = await import("./PostComposer");

beforeEach(() => {
  store.quarantine = null;
});

describe("the post composer", () => {
  it("offers the form to an account that may publish", () => {
    render(<PostComposer communityId="c1" />);
    expect(screen.getByPlaceholderText("Что нового?")).toBeTruthy();
  });

  it("says when publishing opens while the account is held back", () => {
    store.quarantine = { until: "2026-09-29T06:00:00Z", hoursLeft: 18, newChatsPerDay: 20, maxUploadBytes: 2097152 };
    render(<PostComposer communityId="c1" />);
    expect(screen.getByText("Публикация постов откроется через 18 часов")).toBeTruthy();
    expect(screen.queryByPlaceholderText("Что нового?")).toBeNull();
  });
});
