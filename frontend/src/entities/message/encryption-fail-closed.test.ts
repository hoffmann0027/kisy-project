// @vitest-environment node
//
// Audit A-10: any failure to encrypt — a thrown MLS error, a peer with no key
// packages, a session that never started — used to be swallowed, and the
// message went out as plaintext. A private chat is fail-closed: the message is
// NOT sent, and the user is told why in words they can act on.
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Message } from "@shared/api/types";

const api = vi.hoisted(() => ({
  send: vi.fn(),
  forward: vi.fn(),
  schedule: vi.fn(),
}));

const e2ee = vi.hoisted(() => ({
  session: null as unknown,
  encrypt: vi.fn(),
  init: vi.fn(),
}));

vi.mock("@shared/api/endpoints", () => ({
  messagesApi: { send: api.send, forward: api.forward },
  scheduledApi: { schedule: api.schedule },
}));

vi.mock("@entities/e2ee", () => ({
  e2eeSession: () => e2ee.session,
  initE2EE: e2ee.init,
  encryptForChat: e2ee.encrypt,
  cacheOutgoingPlaintext: vi.fn(async () => {}),
  adoptOutgoingPlaintext: vi.fn(async () => {}),
  cachePlaintext: vi.fn(async () => {}),
  cacheScheduledPlaintext: vi.fn(async () => {}),
  cachedScheduledPlaintext: vi.fn(async () => null),
  dropScheduledPlaintext: vi.fn(async () => {}),
  hydrateMessages: vi.fn(async (_s: unknown, m: Message[]) => m),
}));

vi.mock("@shared/store/auth", () => ({
  useAuthStore: { getState: () => ({ user: { id: "user-alice" } }) },
}));

import { userFacingError } from "@shared/api/envelope";
import { forwardMessages, sendMessage } from "./queries";
import { scheduleMessage } from "./scheduled";

const FALLBACK = "__generic__";
const fakeSession = { userId: "user-alice", identity: { deviceId: "dev-a" }, store: {} };
const encryptedBody = { ciphertext: "Y2lwaGVy", alg: 1, epoch: 1 };

function sentMessage(over: Partial<Message> = {}): Message {
  return {
    id: "m-1",
    chatId: "chat-1",
    chatType: "private",
    senderId: "user-alice",
    text: null,
    replyTo: null,
    attachments: [],
    reactions: [],
    mentions: [],
    isDeleted: false,
    createdAt: new Date().toISOString(),
    deletedAt: null,
    editedAt: null,
    pinnedAt: null,
    readCount: null,
    readTotal: null,
    ...over,
  } as Message;
}

async function expectUserFacingRefusal(p: Promise<unknown>) {
  let caught: unknown = null;
  await p.catch((e) => {
    caught = e;
  });
  expect(caught, "the send must fail, not fall back to plaintext").not.toBeNull();
  expect(userFacingError(caught, FALLBACK), "the user must be told why").not.toBe(FALLBACK);
}

beforeEach(() => {
  api.send.mockReset().mockResolvedValue({ message: sentMessage() });
  api.forward.mockReset().mockResolvedValue({ messages: [] });
  api.schedule.mockReset().mockResolvedValue({ scheduled: { id: "s-1" } });
  e2ee.encrypt.mockReset().mockResolvedValue(encryptedBody);
  e2ee.init.mockReset().mockResolvedValue(null);
  e2ee.session = fakeSession;
});

describe("sending into a private chat is fail-closed", () => {
  it("does not send when encryption throws", async () => {
    e2ee.encrypt.mockRejectedValue(new Error("mls: desired gen in the past"));
    await expectUserFacingRefusal(sendMessage("private", "chat-1", "user-bob", { text: "секрет" }));
    expect(api.send).not.toHaveBeenCalled();
  });

  it("does not send when the peer cannot be encrypted for", async () => {
    // What encryptForChat reported for a peer with no key packages.
    e2ee.encrypt.mockResolvedValue(null);
    await expectUserFacingRefusal(sendMessage("private", "chat-1", "user-bob", { text: "секрет" }));
    expect(api.send).not.toHaveBeenCalled();
  });

  it("does not send when the E2EE session is not running, after trying to start it", async () => {
    e2ee.session = null;
    await expectUserFacingRefusal(sendMessage("private", "chat-1", "user-bob", { text: "секрет" }));
    expect(e2ee.init).toHaveBeenCalledWith("user-alice");
    expect(api.send).not.toHaveBeenCalled();
  });

  it("does not send when the peer is unknown", async () => {
    await expectUserFacingRefusal(sendMessage("private", "chat-1", undefined, { text: "секрет" }));
    expect(api.send).not.toHaveBeenCalled();
  });

  it("sends only ciphertext when encryption works", async () => {
    await sendMessage("private", "chat-1", "user-bob", { text: "секрет" });
    expect(api.send).toHaveBeenCalledTimes(1);
    const body = api.send.mock.calls[0][2];
    expect(body.text).toBeUndefined();
    expect(body.ciphertext).toBe(encryptedBody.ciphertext);
  });

  it("still sends plaintext into a group (groups are not end-to-end encrypted yet)", async () => {
    await sendMessage("group", "group-1", undefined, { text: "всем привет" });
    expect(api.send.mock.calls[0][2].text).toBe("всем привет");
  });
});

describe("scheduling into a private chat is fail-closed", () => {
  it("does not schedule when encryption fails", async () => {
    e2ee.encrypt.mockRejectedValue(new Error("network"));
    await expectUserFacingRefusal(
      scheduleMessage("private", "chat-1", "user-bob", { text: "позже", sendAt: new Date(Date.now() + 3600_000) }),
    );
    expect(api.schedule).not.toHaveBeenCalled();
  });
});

describe("forwarding into a private chat is fail-closed", () => {
  const groupMessage = sentMessage({ id: "g-1", chatType: "group", chatId: "group-1", text: "из группы" });
  const target = { chatType: "private" as const, chatId: "chat-1", peerUserId: "user-bob" };

  it("sends the group message's text encrypted, never in the clear", async () => {
    await forwardMessages({ target, messages: [groupMessage], resolveName: () => "alice" });
    expect(api.send).not.toHaveBeenCalled();
    expect(api.forward).toHaveBeenCalledTimes(1);
    const encrypted = api.forward.mock.calls[0][3];
    expect(encrypted?.["g-1"]?.ciphertext).toBe(encryptedBody.ciphertext);
  });

  it("forwards nothing when encryption fails", async () => {
    e2ee.encrypt.mockRejectedValue(new Error("mls"));
    await expectUserFacingRefusal(forwardMessages({ target, messages: [groupMessage], resolveName: () => "alice" }));
    expect(api.forward).not.toHaveBeenCalled();
    expect(api.send).not.toHaveBeenCalled();
  });

  it("forwards nothing when a decrypted E2EE message cannot be re-encrypted", async () => {
    e2ee.encrypt.mockRejectedValue(new Error("mls"));
    const e2eeMessage = sentMessage({ id: "p-1", text: "личное", encrypted: true, chatId: "chat-9" });
    await expectUserFacingRefusal(forwardMessages({ target, messages: [e2eeMessage], resolveName: () => "alice" }));
    expect(api.send).not.toHaveBeenCalled();
  });
});
