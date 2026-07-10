// @vitest-environment node
//
// Manual demo tool (not part of CI): a freshly invited scripted user opens a
// private chat with an EXISTING user and sends one E2EE message. Use it to
// watch a live browser session receive, join and decrypt in real time.
//
//   E2E_BASE_URL=http://localhost:18080/api/v1 \
//   VITE_API_BASE_URL=http://localhost:18080/api/v1 \
//   E2E_CEO_USERNAME=ceo E2E_CEO_PASSWORD=... \
//   E2E_TARGET_USER=ceo E2E_MESSAGE="привет" npx vitest run src/test/e2e/e2ee-send-to-user
import { describe, expect, it } from "vitest";

const BASE = process.env.E2E_BASE_URL ?? "";
const TARGET = process.env.E2E_TARGET_USER ?? "";
const MESSAGE = process.env.E2E_MESSAGE ?? "тестовое зашифрованное сообщение";
const CEO_USER = process.env.E2E_CEO_USERNAME ?? "ceo";
const CEO_PASS = process.env.E2E_CEO_PASSWORD ?? "";

interface Actor {
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

describe.skipIf(!BASE || !TARGET)("send one E2EE message to a live user", () => {
  it("invites a scripted peer and delivers an encrypted message", async () => {
    installFetchJar();
    const api = await import("@shared/api/endpoints");
    const crypto_ = await import("@shared/crypto");
    const e2ee = await import("@entities/e2ee");
    const sessionMod = await import("@entities/e2ee/session");

    // CEO issues an invite and reveals the target's user id.
    const ceo: Actor = { jar: new Map() };
    active = ceo;
    const me = await api.authApi.login(CEO_USER, CEO_PASS);
    const targetId =
      TARGET === CEO_USER
        ? me.user.id
        : (await api.usersApi.directory(TARGET, 5)).users.find((u) => u.username === TARGET)?.id;
    expect(targetId).toBeTruthy();
    const invite = await api.invitesApi.create();

    // The scripted peer registers, brings up an E2EE device and sends.
    const peer: Actor = { jar: new Map() };
    active = peer;
    const username = `e2ep_${Date.now().toString(36)}`;
    const { user } = await api.authApi.register(invite.token, username, "E2eTestPassw0rd!");

    const store = new crypto_.MemoryKeyStore();
    const identity = await crypto_.loadOrCreateIdentity(store);
    const sodium = await crypto_.getSodium();
    await api.e2eeApi.registerDevice({
      deviceId: identity.deviceId,
      name: "demo-peer",
      ed25519Pub: sodium.to_base64(identity.publicKey, sodium.base64_variants.ORIGINAL),
    });
    const session = { store, identity, userId: user.id };
    await api.e2eeApi.uploadKeyPackages(identity.deviceId, await sessionMod.topUpKeyPackages(session, 3));

    // Clearance rules forbid a low-level newcomer from initiating a chat
    // with the CEO — the target opens it, then the peer replies inside it.
    active = ceo;
    const { chat } = await api.chatsApi.open(user.id);
    active = peer;
    const enc = await e2ee.encryptForChat(session, chat.id, targetId!, MESSAGE);
    expect(enc).not.toBeNull();
    const { message } = await api.messagesApi.send("private", chat.id, {
      ciphertext: enc!.ciphertext,
      alg: enc!.alg,
      epoch: enc!.epoch,
      contentKind: 1,
    });
    expect(message.text).toBeNull();
    console.log(`sent encrypted message ${message.id} to chat ${chat.id} as ${username}`);
  }, 60_000);
});
