// @vitest-environment node
//
// End-to-end test of the E2EE orchestration layer against an in-memory fake
// of the server's /e2ee API: alice initiates a private chat, bob joins via
// Welcome, both exchange messages that the "server" only ever sees encrypted.
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ChatType, Message } from "@shared/api/types";

// --- in-memory fake of the server-side E2EE directory/mailbox ---

interface FakeServer {
  keyPackages: Map<string, { deviceId: string; keyPackage: string }[]>; // userId → pool
  welcomes: {
    id: string;
    chatType: ChatType;
    chatId: string;
    recipientDevice: string;
    payload: string;
    acked: boolean;
  }[];
  handshake: { id: string; chatType: ChatType; chatId: string; kind: number; senderDevice: string; payload: string; epoch: number | null; createdAt: string }[];
  deviceOwners: Map<string, string>; // deviceId → userId
}

const server: FakeServer = {
  keyPackages: new Map(),
  welcomes: [],
  handshake: [],
  deviceOwners: new Map(),
};

let nextId = 0;
const genId = () => `srv-${++nextId}`;

vi.mock("@shared/api/endpoints", () => ({
  e2eeApi: {
    async claimKeyPackages(userId: string, excludeDevice?: string) {
      const pool = server.keyPackages.get(userId) ?? [];
      const byDevice = new Map<string, { deviceId: string; keyPackage: string }>();
      for (const kp of pool) {
        if (kp.deviceId === excludeDevice) continue;
        if (!byDevice.has(kp.deviceId)) byDevice.set(kp.deviceId, kp);
      }
      const claimed = [...byDevice.values()];
      server.keyPackages.set(
        userId,
        pool.filter((kp) => !claimed.includes(kp)),
      );
      return { keyPackages: claimed };
    },
    async publishHandshake(body: {
      chatType: ChatType;
      chatId: string;
      kind: "welcome" | "commit" | "proposal";
      senderDevice: string;
      payload: string;
      epoch?: number;
      recipients?: Record<string, string>;
    }) {
      if (body.kind === "welcome") {
        for (const deviceId of Object.keys(body.recipients ?? {})) {
          server.welcomes.push({
            id: genId(),
            chatType: body.chatType,
            chatId: body.chatId,
            recipientDevice: deviceId,
            payload: body.payload,
            acked: false,
          });
        }
      } else {
        server.handshake.push({
          id: genId(),
          chatType: body.chatType,
          chatId: body.chatId,
          kind: body.kind === "commit" ? 2 : 3,
          senderDevice: body.senderDevice,
          payload: body.payload,
          epoch: body.epoch ?? null,
          createdAt: new Date().toISOString(),
        });
      }
      return { published: true };
    },
    async listWelcomes(deviceId: string) {
      return {
        welcomes: server.welcomes
          .filter((w) => w.recipientDevice === deviceId && !w.acked)
          .map((w) => ({
            id: w.id,
            chatType: w.chatType,
            chatId: w.chatId,
            kind: 1,
            senderDevice: null,
            payload: w.payload,
            epoch: null,
            createdAt: new Date().toISOString(),
          })),
      };
    },
    async ackWelcome(welcomeId: string) {
      const w = server.welcomes.find((x) => x.id === welcomeId);
      if (w) w.acked = true;
      return { acked: true };
    },
    async listHandshake(chatType: ChatType, chatId: string, afterId?: string) {
      let items = server.handshake.filter((h) => h.chatType === chatType && h.chatId === chatId);
      if (afterId) {
        const idx = items.findIndex((h) => h.id === afterId);
        items = idx >= 0 ? items.slice(idx + 1) : items;
      }
      return { messages: items };
    },
  },
}));

// Imports below the mock so they see the fake endpoints.
import { MemoryKeyStore, loadOrCreateIdentity } from "@shared/crypto";
import type { E2EESession } from "./session";
import { topUpKeyPackages } from "./session";
import {
  cachePlaintext,
  cachedPlaintext,
  cacheScheduledPlaintext,
  cachedScheduledPlaintext,
  dropPlaintext,
  sweepExpiredPlaintext,
  encryptForChat,
  hydrateMessage,
  processChatHandshake,
  processWelcomes,
  resetChatStatesForTests,
} from "./chats";

async function makeSession(userId: string): Promise<E2EESession> {
  const store = new MemoryKeyStore();
  const identity = await loadOrCreateIdentity(store);
  return { store, identity, userId };
}

/** Register a session's key package pool with the fake server. */
async function publishPool(s: E2EESession, count: number): Promise<void> {
  const fresh = await topUpKeyPackages(s, count);
  const pool = server.keyPackages.get(s.userId) ?? [];
  for (const message of fresh) pool.push({ deviceId: s.identity.deviceId, keyPackage: message });
  server.keyPackages.set(s.userId, pool);
  server.deviceOwners.set(s.identity.deviceId, s.userId);
}

function messageDTO(id: string, chatId: string, senderId: string, ciphertext: string): Message {
  return {
    id,
    chatId,
    chatType: "private",
    senderId,
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
    ciphertext,
    alg: 1,
  };
}

describe("E2EE private chat orchestration", () => {
  beforeEach(() => {
    server.keyPackages.clear();
    server.welcomes.length = 0;
    server.handshake.length = 0;
    server.deviceOwners.clear();
    resetChatStatesForTests();
  });

  it("alice initiates, bob joins via welcome, both directions decrypt", async () => {
    const alice = await makeSession("user-alice");
    const bob = await makeSession("user-bob");
    await publishPool(bob, 3);
    const chatId = "chat-1";

    // Alice sends the first message: the chat group is created on the fly.
    const enc = await encryptForChat(alice, chatId, "user-bob", "привет, боб");
    expect(enc).not.toBeNull();
    expect(enc!.alg).toBe(1);
    // The "server" saw only ciphertext and a welcome for bob's device.
    expect(server.welcomes).toHaveLength(1);

    // Alice caches her own plaintext (send path) and reads it back.
    await cachePlaintext(alice, "m1", "привет, боб");
    const aliceView = await hydrateMessage(alice, messageDTO("m1", chatId, "user-alice", enc!.ciphertext));
    expect(aliceView.text).toBe("привет, боб");
    expect(aliceView.encrypted).toBe(true);

    // Bob joins from the welcome and decrypts the live message.
    const joined = await processWelcomes(bob);
    expect(joined).toEqual([chatId]);
    const bobView = await hydrateMessage(bob, messageDTO("m1", chatId, "user-bob-other", enc!.ciphertext));
    expect(bobView.text).toBe("привет, боб");
    expect(bobView.undecryptable).toBeUndefined();

    // Decryption is one-time; the second hydration hits the plaintext cache.
    const again = await hydrateMessage(bob, messageDTO("m1", chatId, "user-alice", enc!.ciphertext));
    expect(again.text).toBe("привет, боб");

    // Bob replies; alice decrypts.
    const reply = await encryptForChat(bob, chatId, "user-alice", "привет, алиса");
    expect(reply).not.toBeNull();
    const aliceGot = await hydrateMessage(alice, messageDTO("m2", chatId, "user-bob", reply!.ciphertext));
    expect(aliceGot.text).toBe("привет, алиса");

    // The welcome was acknowledged and is not re-delivered.
    expect(await processWelcomes(bob)).toEqual([]);
  });

  it("falls back to plaintext when the peer has no E2EE devices", async () => {
    const alice = await makeSession("user-alice");
    const enc = await encryptForChat(alice, "chat-2", "user-no-devices", "привет");
    expect(enc).toBeNull();
  });

  it("handshake feed skips own commits and stays consistent", async () => {
    const alice = await makeSession("user-alice");
    const bob = await makeSession("user-bob");
    await publishPool(bob, 2);
    const chatId = "chat-3";

    const first = await encryptForChat(alice, chatId, "user-bob", "первое");
    expect(first).not.toBeNull();
    // The initiator's own commit is in the feed; applying it must be a no-op.
    expect(server.handshake).toHaveLength(1);
    await processChatHandshake(alice, "private", chatId);

    const second = await encryptForChat(alice, chatId, "user-bob", "второе");
    expect(second).not.toBeNull();

    await processWelcomes(bob);
    const got1 = await hydrateMessage(bob, messageDTO("h1", chatId, "user-alice", first!.ciphertext));
    const got2 = await hydrateMessage(bob, messageDTO("h2", chatId, "user-alice", second!.ciphertext));
    expect(got1.text).toBe("первое");
    expect(got2.text).toBe("второе");
  });

  it("adopts a scheduled plaintext onto the delivered message id (stage I)", async () => {
    const alice = await makeSession("user-alice");
    const bob = await makeSession("user-bob");
    await publishPool(bob, 2);
    const chatId = "chat-sched";

    // Scheduling: alice encrypts now, the plaintext is cached under the
    // scheduled id (she cannot decrypt her own ciphertext later).
    const enc = await encryptForChat(alice, chatId, "user-bob", "отложенное");
    expect(enc).not.toBeNull();
    await cacheScheduledPlaintext(alice, "sched-1", "отложенное");

    // Delivery: the worker's message arrives with scheduledId set — the
    // cached plaintext is re-keyed onto the real message id.
    const delivered = { ...messageDTO("m-real", chatId, "user-alice", enc!.ciphertext), scheduledId: "sched-1" };
    const view = await hydrateMessage(alice, delivered);
    expect(view.text).toBe("отложенное");
    expect(view.encrypted).toBe(true);

    // The sched entry is gone; the msg entry answers future hydrations.
    expect(await cachedScheduledPlaintext(alice, "sched-1")).toBeNull();
    const again = await hydrateMessage(alice, delivered);
    expect(again.text).toBe("отложенное");
  });

  it("purges the plaintext cache when a message disappears (stage J)", async () => {
    const alice = await makeSession("user-alice");
    const bob = await makeSession("user-bob");
    await publishPool(bob, 2);
    const chatId = "chat-ttl";

    const enc = await encryptForChat(alice, chatId, "user-bob", "самоуничтожаюсь");
    expect(enc).not.toBeNull();
    await cachePlaintext(alice, "m-ttl", "самоуничтожаюсь");

    // Readable while alive.
    const alive = await hydrateMessage(alice, messageDTO("m-ttl", chatId, "user-alice", enc!.ciphertext));
    expect(alive.text).toBe("самоуничтожаюсь");

    // message.deleted arrives → the local plaintext MUST be purged. The
    // ciphertext is undecryptable afterwards (one-time MLS keys), so the
    // disappeared text is unrecoverable on this device.
    await dropPlaintext(alice, "m-ttl");
    const gone = await hydrateMessage(alice, messageDTO("m-ttl", chatId, "user-alice", enc!.ciphertext));
    expect(gone.text).toBeNull();
    expect(gone.undecryptable).toBe(true);
  });

  it("self-evicts expired plaintext even without a delete event (stage J offline gap)", async () => {
    const alice = await makeSession("user-alice");
    const past = new Date(Date.now() - 60_000).toISOString();
    const future = new Date(Date.now() + 60 * 60_000).toISOString();

    // A message that expired while this device was offline, and one still live.
    await cachePlaintext(alice, "expired", "исчезло пока был офлайн", past);
    await cachePlaintext(alice, "alive", "ещё живое", future);

    // Reading the expired entry enforces the timer and evicts it.
    expect(await cachedPlaintext(alice, "expired")).toBeNull();
    expect(await cachedPlaintext(alice, "alive")).toBe("ещё живое");

    // The periodic sweep removes expired entries proactively (no read needed).
    await cachePlaintext(alice, "expired2", "тоже истекло", past);
    const removed = await sweepExpiredPlaintext(alice);
    expect(removed).toBe(1);
    expect(await cachedPlaintext(alice, "expired2")).toBeNull();
    // Non-expiring and future entries survive the sweep.
    expect(await cachedPlaintext(alice, "alive")).toBe("ещё живое");
  });

  it("marks history from before the device joined as undecryptable", async () => {
    const alice = await makeSession("user-alice");
    const bob = await makeSession("user-bob");
    await publishPool(bob, 2);
    const chatId = "chat-4";

    // Alice encrypts BEFORE bob exists in the group state she later rebuilds:
    // simulate by encrypting in a chat bob never joined.
    const enc = await encryptForChat(alice, chatId, "user-bob", "старая история");
    expect(enc).not.toBeNull();

    // A third device without any state sees a lock, not a crash.
    const stranger = await makeSession("user-charlie");
    const view = await hydrateMessage(stranger, messageDTO("m9", chatId, "user-alice", enc!.ciphertext));
    expect(view.undecryptable).toBe(true);
    expect(view.text).toBeNull();
  });
});
