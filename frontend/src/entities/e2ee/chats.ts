// Per-chat MLS state: creation on first send, joining via Welcome, message
// encrypt/decrypt and the local plaintext cache (docs/e2ee-design.md §5.2 —
// MLS keys are one-time, so server history is undecryptable later; decrypted
// text must be cached locally, encrypted at rest by the keystore).
import {
  addMembers,
  createChat,
  createDeviceKeyPackage,
  deserializeChatState,
  encryptMessage,
  getSodium,
  joinChat,
  processIncoming,
  serializeChatState,
  KISY_E2EE_ALG,
  type ChatState,
} from "@shared/crypto";
import { e2eeApi } from "@shared/api/endpoints";
import type { ChatType, Message } from "@shared/api/types";
import { dropKeyPackage, localKeyPackages, type E2EESession } from "./session";

const utf8 = (t: string) => new TextEncoder().encode(t);
const utf8dec = new TextDecoder();

// --- per-chat serialization: MLS state must never be mutated concurrently ---

const locks = new Map<string, Promise<unknown>>();

function withChatLock<T>(chatId: string, fn: () => Promise<T>): Promise<T> {
  const prev = locks.get(chatId) ?? Promise.resolve();
  const next = prev.then(fn, fn);
  locks.set(
    chatId,
    next.catch(() => undefined),
  );
  return next;
}

// --- state persistence ---
//
// The in-memory map is keyed by device AND chat: a browser can host several
// accounts over time (or tests several sessions), and one session's state
// must never bleed into another's.

const states = new Map<string, ChatState>();

function stateKey(s: E2EESession, chatId: string): string {
  return `${s.identity.deviceId}/${chatId}`;
}

async function loadState(s: E2EESession, chatId: string): Promise<ChatState | null> {
  const cached = states.get(stateKey(s, chatId));
  if (cached) return cached;
  const raw = await s.store.get(`mls/private/${chatId}`);
  if (!raw) return null;
  const state = deserializeChatState(raw);
  states.set(stateKey(s, chatId), state);
  return state;
}

async function saveState(s: E2EESession, chatId: string, state: ChatState): Promise<void> {
  states.set(stateKey(s, chatId), state);
  await s.store.put(`mls/private/${chatId}`, serializeChatState(state));
}

/** Test-only: clear in-memory chat state. */
export function resetChatStatesForTests(): void {
  states.clear();
  locks.clear();
}

// --- plaintext cache (messageId → decrypted text) ---

export async function cachePlaintext(s: E2EESession, messageId: string, text: string): Promise<void> {
  await s.store.put(`msg/${messageId}`, utf8(text));
}

export async function cachedPlaintext(s: E2EESession, messageId: string): Promise<string | null> {
  const raw = await s.store.get(`msg/${messageId}`);
  return raw ? utf8dec.decode(raw) : null;
}

/**
 * Purge a message's locally cached plaintext (stage J). MUST be called for
 * every message.deleted event — otherwise a "disappeared" E2EE message
 * would silently survive in IndexedDB and the security feature would leak.
 * Also the hook for the future client-side search index (E2EE stage 8):
 * add its cleanup here.
 */
export async function dropPlaintext(s: E2EESession, messageId: string): Promise<void> {
  await s.store.remove(`msg/${messageId}`);
}

// Scheduled sending (stage I): the client encrypts at scheduling time, so
// the plaintext is cached under the scheduled id until the real message
// exists — MLS senders consume their keys at encryption time and cannot
// decrypt their own ciphertext later.

export async function cacheScheduledPlaintext(s: E2EESession, scheduledId: string, text: string): Promise<void> {
  await s.store.put(`sched/${scheduledId}`, utf8(text));
}

export async function cachedScheduledPlaintext(s: E2EESession, scheduledId: string): Promise<string | null> {
  const raw = await s.store.get(`sched/${scheduledId}`);
  return raw ? utf8dec.decode(raw) : null;
}

export async function dropScheduledPlaintext(s: E2EESession, scheduledId: string): Promise<void> {
  await s.store.remove(`sched/${scheduledId}`);
}

/** Re-key a scheduled plaintext onto the delivered message id. */
async function adoptScheduledPlaintext(s: E2EESession, scheduledId: string, messageId: string): Promise<string | null> {
  const text = await cachedScheduledPlaintext(s, scheduledId);
  if (text === null) return null;
  await cachePlaintext(s, messageId, text);
  await dropScheduledPlaintext(s, scheduledId);
  return text;
}

// --- welcomes: join chats other devices invited us into ---

/**
 * Fetch pending Welcomes for this device and join the corresponding chats.
 * Returns the chat ids that gained a state.
 */
export async function processWelcomes(s: E2EESession): Promise<string[]> {
  const sodium = await getSodium();
  const { welcomes } = await e2eeApi.listWelcomes(s.identity.deviceId);
  if (welcomes.length === 0) return [];

  const pool = await localKeyPackages(s);
  const joined: string[] = [];

  for (const w of welcomes) {
    const joinedChat = await withChatLock(w.chatId, async () => {
      // A state can already exist if both sides initiated simultaneously; the
      // existing state wins and the stray welcome is acknowledged away
      // (rare race, documented in docs/e2ee-design.md).
      if (await loadState(s, w.chatId)) return false;
      const payload = sodium.from_base64(w.payload, sodium.base64_variants.ORIGINAL);
      let lastError: unknown = null;
      for (const { n, pkg } of pool) {
        try {
          const state = await joinChat(payload, pkg);
          await saveState(s, w.chatId, state);
          await dropKeyPackage(s, n);
          return true;
        } catch (err) {
          // Welcome was addressed to a different key package — try the next.
          lastError = err;
        }
      }
      console.warn(`E2EE: welcome ${w.id} for chat ${w.chatId} did not open with any local key package`, lastError);
      return false;
    });
    // Always ack: either joined, redundant, or permanently unopenable
    // (its key package is gone) — re-delivery would never succeed.
    await e2eeApi.ackWelcome(w.id, s.identity.deviceId).catch(() => {});
    if (joinedChat) joined.push(w.chatId);
  }
  return joined;
}

// --- handshake feed: apply commits from other members/devices ---

export async function processChatHandshake(s: E2EESession, chatType: ChatType, chatId: string): Promise<void> {
  if (chatType !== "private") return; // groups are stage 5
  await withChatLock(chatId, async () => {
    let state = await loadState(s, chatId);
    if (!state) return;

    const sodium = await getSodium();
    const cursorRaw = await s.store.get(`hs/${chatId}`);
    const afterId = cursorRaw ? utf8dec.decode(cursorRaw) : undefined;
    const { messages } = await e2eeApi.listHandshake(chatType, chatId, afterId);

    for (const frame of messages) {
      // Our own commits are already applied locally at creation time — MLS
      // cannot process a commit authored by itself.
      if (frame.senderDevice !== s.identity.deviceId) {
        try {
          const result = await processIncoming(
            state,
            sodium.from_base64(frame.payload, sodium.base64_variants.ORIGINAL),
          );
          state = result.state;
        } catch (err) {
          console.warn(`E2EE: failed to apply handshake ${frame.id} for chat ${chatId}`, err);
        }
      }
      await s.store.put(`hs/${chatId}`, utf8(frame.id));
    }
    await saveState(s, chatId, state);
  });
}

// --- chat creation (first E2EE message in a private chat) ---

async function initiateChat(s: E2EESession, chatId: string, peerUserId: string): Promise<ChatState | null> {
  const sodium = await getSodium();
  const fromB64 = (t: string) => sodium.from_base64(t, sodium.base64_variants.ORIGINAL);
  const toB64 = (u: Uint8Array) => sodium.to_base64(u, sodium.base64_variants.ORIGINAL);

  // One key package per device: the peer's devices + our own other devices.
  const [peer, ownOthers] = await Promise.all([
    e2eeApi.claimKeyPackages(peerUserId),
    e2eeApi.claimKeyPackages(s.userId, s.identity.deviceId),
  ]);
  if (peer.keyPackages.length === 0) {
    // Peer has no E2EE devices yet (never logged in since the rollout) —
    // fall back to plaintext for now; retried on the next send.
    return null;
  }

  const recipients: Record<string, string> = {};
  for (const kp of peer.keyPackages) recipients[kp.deviceId] = peerUserId;
  for (const kp of ownOthers.keyPackages) recipients[kp.deviceId] = s.userId;
  const packages = [...peer.keyPackages, ...ownOthers.keyPackages].map((kp) => fromB64(kp.keyPackage));

  const ownPackage = await createDeviceKeyPackage(s.identity, s.userId);
  let state = await createChat(`private/${chatId}`, ownPackage);
  const commit = await addMembers(state, packages);
  state = commit.state;

  if (commit.welcome) {
    await e2eeApi.publishHandshake({
      chatType: "private",
      chatId,
      kind: "welcome",
      senderDevice: s.identity.deviceId,
      payload: toB64(commit.welcome),
      epoch: Number(commit.epoch),
      recipients,
    });
  }
  // The commit is published for completeness/epoch tracking; other members
  // join from the Welcome and skip frames authored by this device.
  await e2eeApi.publishHandshake({
    chatType: "private",
    chatId,
    kind: "commit",
    senderDevice: s.identity.deviceId,
    payload: toB64(commit.commit),
    epoch: Number(commit.epoch),
  });

  await saveState(s, chatId, state);
  return state;
}

// --- public API: encrypt / decrypt ---

export interface EncryptedBody {
  ciphertext: string; // base64
  alg: number;
  epoch: number;
}

/**
 * Encrypt a message for a private chat, creating the chat's MLS group on
 * first use. Returns null when E2EE is not possible yet (peer has no devices)
 * — the caller falls back to plaintext.
 */
export async function encryptForChat(
  s: E2EESession,
  chatId: string,
  peerUserId: string,
  text: string,
): Promise<EncryptedBody | null> {
  return withChatLock(chatId, async () => {
    let state = await loadState(s, chatId);
    if (!state) state = await initiateChat(s, chatId, peerUserId);
    if (!state) return null;

    const sodium = await getSodium();
    const result = await encryptMessage(state, utf8(text));
    await saveState(s, chatId, result.state);
    return {
      ciphertext: sodium.to_base64(result.message, sodium.base64_variants.ORIGINAL),
      alg: KISY_E2EE_ALG,
      epoch: Number(result.epoch),
    };
  });
}

/**
 * Decrypt one incoming private-chat message. Consumes the one-time MLS key,
 * so the result is immediately cached; returns null when undecryptable
 * (e.g. history from before this device joined).
 */
async function decryptOnce(s: E2EESession, chatId: string, messageId: string, ciphertextB64: string): Promise<string | null> {
  return withChatLock(chatId, async () => {
    const state = await loadState(s, chatId);
    if (!state) return null;
    const sodium = await getSodium();
    try {
      const result = await processIncoming(
        state,
        sodium.from_base64(ciphertextB64, sodium.base64_variants.ORIGINAL),
      );
      await saveState(s, chatId, result.state);
      if (result.kind !== "message") return null;
      const text = utf8dec.decode(result.plaintext);
      await cachePlaintext(s, messageId, text);
      return text;
    } catch {
      return null;
    }
  });
}

/**
 * Resolve a message DTO for display: plaintext messages pass through;
 * encrypted ones get their text from the local cache or a one-time live
 * decryption. Undecryptable messages are flagged for the UI.
 */
export async function hydrateMessage(s: E2EESession, msg: Message): Promise<Message> {
  if (!msg.ciphertext || msg.isDeleted) return msg;

  const cached = await cachedPlaintext(s, msg.id);
  if (cached !== null) return { ...msg, text: cached, encrypted: true };

  // A message born from the scheduler: its plaintext was cached under the
  // scheduled id at scheduling time (the sender cannot decrypt its own
  // ciphertext) — adopt it onto the real message id.
  if (msg.scheduledId) {
    const adopted = await adoptScheduledPlaintext(s, msg.scheduledId, msg.id);
    if (adopted !== null) return { ...msg, text: adopted, encrypted: true };
  }

  if (msg.chatType === "private") {
    // A fresh ciphertext we have not seen: try the live decryption path.
    // Our own outgoing messages cannot be decrypted (MLS senders consume
    // their keys at encryption time) — their plaintext is cached by the
    // send path; a miss here means another of our devices sent it.
    const text = await decryptOnce(s, msg.chatId, msg.id, msg.ciphertext);
    if (text !== null) return { ...msg, text, encrypted: true };
  }
  return { ...msg, text: null, encrypted: true, undecryptable: true };
}

export async function hydrateMessages(s: E2EESession, msgs: Message[]): Promise<Message[]> {
  // Oldest-first so live decryption ratchets forward in message order.
  const byAge = [...msgs].sort((a, b) => (a.createdAt < b.createdAt ? -1 : 1));
  const hydrated = new Map<string, Message>();
  for (const m of byAge) {
    hydrated.set(m.id, await hydrateMessage(s, m));
  }
  return msgs.map((m) => hydrated.get(m.id) ?? m);
}
