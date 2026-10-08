import { afterEach, describe, expect, it, vi } from "vitest";
import { closeAnnouncementPush, closeWithdrawnAnnouncementPushes } from "./shownNotifications";

// An announcement taken back stayed on screen as a push. These are the browser
// half of taking it down: close exactly the revoked one, never a chat push.

function shown(...tags: string[]) {
  const notes = tags.map((tag) => ({ tag, close: vi.fn() }));
  Object.defineProperty(navigator, "serviceWorker", {
    configurable: true,
    value: { getRegistration: async () => ({ getNotifications: async () => notes }) },
  });
  return notes;
}

afterEach(() => {
  Object.defineProperty(navigator, "serviceWorker", { configurable: true, value: undefined });
});

describe("taking an announcement's push down", () => {
  it("closes the revoked announcement and nothing else", async () => {
    const [revoked, other, chat] = shown("announcement-a1", "announcement-a2", "kisy");
    await closeAnnouncementPush("a1");
    expect(revoked.close).toHaveBeenCalled();
    expect(other.close).not.toHaveBeenCalled();
    expect(chat.close).not.toHaveBeenCalled();
  });

  it("closes what was revoked while the app was closed", async () => {
    const [kept, gone, chat] = shown("announcement-a1", "announcement-a2", "kisy");
    await closeWithdrawnAnnouncementPushes(["a1"]);
    expect(kept.close).not.toHaveBeenCalled();
    expect(gone.close).toHaveBeenCalled();
    expect(chat.close).not.toHaveBeenCalled();
  });
});
