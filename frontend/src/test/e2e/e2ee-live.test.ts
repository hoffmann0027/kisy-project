// @vitest-environment node
//
// Live E2EE proof against a RUNNING backend (stage 10 §5 groundwork):
// two scripted devices (built on the real shared/crypto + entities/e2ee
// code) register, exchange an MLS-encrypted conversation through the real
// API and verify the server never returns (or stores) plaintext.
//
// Skipped unless E2E_BASE_URL is set, e.g.:
//   E2E_BASE_URL=http://localhost:18080/api/v1 \
//   E2E_CEO_USERNAME=ceo E2E_CEO_PASSWORD=... npx vitest run src/test/e2e
//
// (VITE_API_BASE_URL must point at the same URL so the api client uses it.)
import { beforeAll, describe, expect, it } from "vitest";

const BASE = process.env.E2E_BASE_URL ?? "";
const CEO_USER = process.env.E2E_CEO_USERNAME ?? "ceo";
const CEO_PASS = process.env.E2E_CEO_PASSWORD ?? "";

// --- per-actor cookie jar over global fetch ---
// The api client and entities/e2ee are browser modules with ambient cookie
// auth; in Node we emulate the browser by injecting the active actor's
// cookies into every request and capturing Set-Cookie responses.

interface Actor {
  name: string;
  jar: Map<string, string>;
}

let active: Actor | null = null;

function installFetchJar() {
  const real = globalThis.fetch;
  globalThis.fetch = async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const headers = new Headers(init.headers);
    if (active && active.jar.size > 0) {
      headers.set("cookie", [...active.jar.entries()].map(([k, v]) => `${k}=${v}`).join("; "));
    }
    const res = await real(input, { ...init, headers });
    for (const line of res.headers.getSetCookie()) {
      const [pair] = line.split(";");
      const eq = pair.indexOf("=");
      if (eq > 0 && active) active.jar.set(pair.slice(0, eq).trim(), pair.slice(eq + 1));
    }
    return res;
  };
}

const as = (a: Actor) => {
  active = a;
};

describe.skipIf(!BASE)("E2EE against a live server", () => {
  // Imported lazily so import.meta.env (VITE_API_BASE_URL) and the fetch
  // wrapper are in place before the api client module loads.
  let api: typeof import("@shared/api/endpoints");
  let crypto_: typeof import("@shared/crypto");
  let e2ee: typeof import("@entities/e2ee");
  let sessionMod: typeof import("@entities/e2ee/session");

  const suffix = Date.now().toString(36);
  const ceo: Actor = { name: "ceo", jar: new Map() };
  const alice: Actor = { name: `e2ea_${suffix}`, jar: new Map() };
  const bob: Actor = { name: `e2eb_${suffix}`, jar: new Map() };
  const PASS = "E2eTestPassw0rd!";

  let aliceId = "";
  let bobId = "";
  let chatId = "";
  let aliceSession: import("@entities/e2ee").E2EESession;
  let bobSession: import("@entities/e2ee").E2EESession;

  beforeAll(async () => {
    installFetchJar();
    api = await import("@shared/api/endpoints");
    crypto_ = await import("@shared/crypto");
    e2ee = await import("@entities/e2ee");
    sessionMod = await import("@entities/e2ee/session");

    // CEO invites two throwaway users.
    as(ceo);
    await api.authApi.login(CEO_USER, CEO_PASS);
    const inviteA = await api.invitesApi.create();
    const inviteB = await api.invitesApi.create();

    as(alice);
    aliceId = (await api.authApi.register(inviteA.token, alice.name, PASS)).user.id;
    as(bob);
    bobId = (await api.authApi.register(inviteB.token, bob.name, PASS)).user.id;
  }, 60_000);

  async function makeDevice(userId: string): Promise<import("@entities/e2ee").E2EESession> {
    const store = new crypto_.MemoryKeyStore();
    const identity = await crypto_.loadOrCreateIdentity(store);
    const sodium = await crypto_.getSodium();
    await api.e2eeApi.registerDevice({
      deviceId: identity.deviceId,
      name: "e2e-script",
      ed25519Pub: sodium.to_base64(identity.publicKey, sodium.base64_variants.ORIGINAL),
    });
    const session = { store, identity, userId };
    const fresh = await sessionMod.topUpKeyPackages(session, 5);
    await api.e2eeApi.uploadKeyPackages(identity.deviceId, fresh);
    return session;
  }

  it("registers devices and key packages in the directory", async () => {
    as(alice);
    aliceSession = await makeDevice(aliceId);
    as(bob);
    bobSession = await makeDevice(bobId);

    const { devices } = await api.e2eeApi.listDevices(aliceId);
    expect(devices.map((d) => d.id)).toContain(aliceSession.identity.deviceId);
    const { available } = await api.e2eeApi.countKeyPackages(bobSession.identity.deviceId);
    expect(available).toBe(5);
  }, 60_000);

  it("exchanges an encrypted conversation the server cannot read", async () => {
    // Alice opens the chat and sends the first encrypted message.
    as(alice);
    const { chat } = await api.chatsApi.open(bobId);
    chatId = chat.id;

    const secret = `совершенно секретно ${suffix}`;
    const enc = await e2ee.encryptForChat(aliceSession, chatId, bobId, secret);
    expect(enc).not.toBeNull();
    const { message: sent } = await api.messagesApi.send("private", chatId, {
      ciphertext: enc!.ciphertext,
      alg: enc!.alg,
      epoch: enc!.epoch,
      contentKind: 1,
    });
    expect(sent.text).toBeNull();

    // What the server returns to ANY reader is ciphertext, not the secret.
    const page = await api.messagesApi.list("private", chatId);
    const row = page.items.find((m) => m.id === sent.id)!;
    expect(row.text).toBeNull();
    expect(row.ciphertext).toBeTruthy();
    expect(row.ciphertext).not.toContain(secret);
    expect(atob(row.ciphertext!)).not.toContain(secret);

    // Bob joins via the Welcome and reads the message.
    as(bob);
    const joined = await e2ee.processWelcomes(bobSession);
    expect(joined).toContain(chatId);
    const bobView = await e2ee.hydrateMessage(bobSession, row);
    expect(bobView.text).toBe(secret);

    // Bob replies; alice decrypts the reply fetched from the server.
    const replyText = `ответ ${suffix}`;
    const replyEnc = await e2ee.encryptForChat(bobSession, chatId, aliceId, replyText);
    const { message: reply } = await api.messagesApi.send("private", chatId, {
      ciphertext: replyEnc!.ciphertext,
      alg: replyEnc!.alg,
      epoch: replyEnc!.epoch,
      contentKind: 1,
    });

    as(alice);
    const page2 = await api.messagesApi.list("private", chatId);
    const replyRow = page2.items.find((m) => m.id === reply.id)!;
    expect(replyRow.text).toBeNull();
    const aliceView = await e2ee.hydrateMessage(aliceSession, replyRow);
    expect(aliceView.text).toBe(replyText);
  }, 60_000);
});
