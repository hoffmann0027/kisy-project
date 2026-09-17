// @vitest-environment node
import { describe, expect, it } from "vitest";
import type { Message } from "@shared/api/types";
import { canEditMessage } from "./permissions";

const base = { id: "m", chatId: "c", senderId: "me", text: "hi" } as Message;

describe("canEditMessage", () => {
  it("offers edit on the viewer's own group message", () => {
    expect(canEditMessage({ ...base, chatType: "group" }, "me")).toBe(true);
  });

  it("never offers edit in a private chat, even on a legacy plaintext row (A-35)", () => {
    expect(canEditMessage({ ...base, chatType: "private" }, "me")).toBe(false);
    expect(canEditMessage({ ...base, chatType: "private", encrypted: true }, "me")).toBe(false);
  });

  it("never offers edit on someone else's message", () => {
    expect(canEditMessage({ ...base, chatType: "group" }, "other")).toBe(false);
  });
});
