import { fireEvent, render, screen } from "@testing-library/react";
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

// Feedback from a user: the bar of actions lit up on every message the
// cursor crossed. It opens on a click on the message now, and goes away after
// an action, a click elsewhere or Esc.
describe("MessageBubble action bar", () => {
  it("does not open on hover, opens on a click", () => {
    renderBubble({ text: "привет" });
    fireEvent.mouseEnter(screen.getByText("привет"));
    fireEvent.mouseOver(screen.getByText("привет"));
    expect(screen.queryByTitle("Ответить")).toBeNull();

    fireEvent.click(screen.getByText("привет"));
    expect(screen.getByTitle("Ответить")).toBeTruthy();
  });

  it("runs the action and puts the bar away", () => {
    const onReply = vi.fn();
    renderBubble({ text: "привет" }, { onReply });
    fireEvent.click(screen.getByText("привет"));
    fireEvent.click(screen.getByTitle("Ответить"));
    expect(onReply).toHaveBeenCalledTimes(1);
    expect(screen.queryByTitle("Ответить")).toBeNull();
  });

  it("closes on a click elsewhere and on Esc", () => {
    renderBubble({ text: "привет" });
    fireEvent.click(screen.getByText("привет"));
    fireEvent.mouseDown(document.body);
    expect(screen.queryByTitle("Ответить")).toBeNull();

    fireEvent.click(screen.getByText("привет"));
    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByTitle("Ответить")).toBeNull();
  });

  it("leaves a photo's click to the photo", () => {
    const onOpenImage = vi.fn();
    renderBubble(
      {
        text: "смотри",
        attachments: [{ id: "a1", fileName: "pic.png", url: "/api/v1/attachments/a1", sizeBytes: 10, mimeType: "image/png", isImage: true }],
      } as Partial<Message>,
      { onOpenImage },
    );
    fireEvent.click(screen.getByTitle("pic.png"));
    expect(onOpenImage).toHaveBeenCalledTimes(1);
    expect(screen.queryByTitle("Ответить")).toBeNull();
  });

  // The picker sat inside the chat's scroll area: near the top of a chat it
  // was cut off, and focusing its search scrolled the whole chat sideways.
  it("floats the emoji picker at the page root", () => {
    const { container } = renderBubble({ text: "привет" });
    fireEvent.click(screen.getByText("привет"));
    fireEvent.click(screen.getByTitle("Больше эмодзи"));
    const picker = screen.getByRole("dialog", { name: "Выбор эмодзи" });
    expect(picker.classList.contains("emojipick--floating")).toBe(true);
    expect(container.contains(picker)).toBe(false);
    // Picking in it does not count as a click elsewhere.
    fireEvent.mouseDown(picker);
    expect(screen.getByTitle("Ответить")).toBeTruthy();
  });
});
