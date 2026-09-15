import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Message } from "@shared/api/types";
import { MessageBubble } from "./MessageBubble";

// The padlock used to be decided in two places that disagreed: the compact
// selection row preferred the text, the bubble preferred the undecryptable
// flag. A message carrying both — text recovered from the local cache plus a
// flag set when it first arrived — rendered its text in one place and
// "🔒 Зашифрованное сообщение" in the other, at the same time. This is the
// rule that replaced them: text wins, always.

const LOCK = /Зашифрованное сообщение/;

function message(over: Partial<Message>): Message {
  return {
    id: "m1",
    chatId: "c1",
    chatType: "private",
    senderId: "u1",
    text: null,
    createdAt: new Date().toISOString(),
    editedAt: null,
    isDeleted: false,
    attachments: [],
    reactions: [],
    mentions: [],
    ...over,
  } as Message;
}

function renderBubble(over: Partial<Message>, extra: Record<string, unknown> = {}) {
  const noop = vi.fn();
  return render(
    <MessageBubble
      message={message(over)}
      mine
      canDelete={false}
      canEdit={false}
      onReply={noop}
      onEdit={noop}
      onDelete={noop}
      onReact={noop}
      onPin={noop}
      onOpenImage={noop}
      onForward={noop}
      {...extra}
    />,
  );
}

describe("MessageBubble padlock", () => {
  it("shows the padlock only when there is no text to show", () => {
    renderBubble({ text: null, undecryptable: true, encrypted: true });
    expect(screen.getByText(LOCK)).toBeTruthy();
  });

  it("shows the text, not the padlock, when both are present", () => {
    renderBubble({ text: "Как же руки чешутся", undecryptable: true, encrypted: true });
    expect(screen.getByText("Как же руки чешутся")).toBeTruthy();
    expect(screen.queryByText(LOCK)).toBeNull();
  });

  it("agrees with itself in the selection row", () => {
    // The second render path — the one that used to disagree.
    renderBubble(
      { text: "Как же руки чешутся", undecryptable: true, encrypted: true },
      { selectionMode: true, selected: false, onToggleSelect: vi.fn() },
    );
    expect(screen.getByText("Как же руки чешутся")).toBeTruthy();
    expect(screen.queryByText(LOCK)).toBeNull();
  });

  it("never locks a message that has text and an attachment", () => {
    renderBubble({
      text: "смотри",
      undecryptable: true,
      encrypted: true,
      attachments: [
        {
          id: "a1",
          fileName: "pic.png",
          url: "/api/v1/attachments/a1",
          sizeBytes: 10,
          mimeType: "image/png",
          isImage: true,
        },
      ],
    } as Partial<Message>);
    expect(screen.getByText("смотри")).toBeTruthy();
    expect(screen.queryByText(LOCK)).toBeNull();
  });
});
