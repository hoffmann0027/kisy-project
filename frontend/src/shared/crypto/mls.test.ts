// @vitest-environment node
import { describe, expect, it } from "vitest";
import { MemoryKeyStore } from "./keystore";
import { loadOrCreateIdentity, type DeviceIdentity } from "./identity";
import {
  addMembers,
  createChat,
  createDeviceKeyPackage,
  currentEpoch,
  deserializeChatState,
  encryptMessage,
  joinChat,
  listMembers,
  processIncoming,
  removeMember,
  selfUpdate,
  serializeChatState,
  type ChatState,
  type DeviceKeyPackage,
} from "./mls";

interface TestUser {
  userId: string;
  identity: DeviceIdentity;
  keyPackage: DeviceKeyPackage;
}

async function makeUser(userId: string): Promise<TestUser> {
  const identity = await loadOrCreateIdentity(new MemoryKeyStore());
  return { userId, identity, keyPackage: await createDeviceKeyPackage(identity, userId) };
}

function utf8(text: string): Uint8Array {
  return new TextEncoder().encode(text);
}

function decode(bytes: Uint8Array): string {
  return new TextDecoder().decode(bytes);
}

async function expectMessage(state: ChatState, bytes: Uint8Array): Promise<{ state: ChatState; text: string }> {
  const result = await processIncoming(state, bytes);
  if (result.kind !== "message") throw new Error("Expected application message");
  return { state: result.state, text: decode(result.plaintext) };
}

describe("MLS chat (1:1 as a 2-member group)", () => {
  it("full flow: create, add, both directions, epochs advance", async () => {
    const alice = await makeUser("alice");
    const bob = await makeUser("bob");

    let aliceState = await createChat("chat-1", alice.keyPackage);
    const added = await addMembers(aliceState, [bob.keyPackage.keyPackageMessage]);
    aliceState = added.state;
    expect(added.welcome).not.toBeNull();
    expect(currentEpoch(aliceState)).toBe(1n);

    let bobState = await joinChat(added.welcome!, bob.keyPackage);
    expect(listMembers(bobState).map((m) => m.userId).sort()).toEqual(["alice", "bob"]);

    // alice → bob
    const fromAlice = await encryptMessage(aliceState, utf8("привет, боб"));
    aliceState = fromAlice.state;
    const bobGot = await expectMessage(bobState, fromAlice.message);
    bobState = bobGot.state;
    expect(bobGot.text).toBe("привет, боб");

    // bob → alice
    const fromBob = await encryptMessage(bobState, utf8("привет, алиса"));
    bobState = fromBob.state;
    const aliceGot = await expectMessage(aliceState, fromBob.message);
    expect(aliceGot.text).toBe("привет, алиса");
  });

  it("ciphertext does not contain the plaintext (server-blindness proof)", async () => {
    const alice = await makeUser("alice");
    const bob = await makeUser("bob");

    let state = await createChat("chat-2", alice.keyPackage);
    const added = await addMembers(state, [bob.keyPackage.keyPackageMessage]);
    state = added.state;

    const secret = "СОВЕРШЕННО_СЕКРЕТНО_top-secret-42";
    const { message } = await encryptMessage(state, utf8(secret));

    // Neither raw bytes nor any text decoding of them contain the plaintext.
    const asLatin1 = Array.from(message, (b) => String.fromCharCode(b)).join("");
    expect(asLatin1).not.toContain(secret);
    expect(new TextDecoder("utf-8", { fatal: false }).decode(message)).not.toContain(secret);
  });

  it("group state survives serialize/deserialize and keeps working", async () => {
    const alice = await makeUser("alice");
    const bob = await makeUser("bob");

    let aliceState = await createChat("chat-3", alice.keyPackage);
    const added = await addMembers(aliceState, [bob.keyPackage.keyPackageMessage]);
    aliceState = added.state;
    let bobState = await joinChat(added.welcome!, bob.keyPackage);

    // Simulate app restart on alice's side.
    aliceState = deserializeChatState(serializeChatState(aliceState));

    const msg = await encryptMessage(aliceState, utf8("после перезапуска"));
    const got = await expectMessage(bobState, msg.message);
    expect(got.text).toBe("после перезапуска");
  });
});

describe("MLS group membership (RBAC hook)", () => {
  it("removed member cannot read messages of the new epoch", async () => {
    const alice = await makeUser("alice");
    const bob = await makeUser("bob");
    const eve = await makeUser("eve");

    // Group of three.
    let aliceState = await createChat("group-1", alice.keyPackage);
    const added = await addMembers(aliceState, [
      bob.keyPackage.keyPackageMessage,
      eve.keyPackage.keyPackageMessage,
    ]);
    aliceState = added.state;
    let bobState = await joinChat(added.welcome!, bob.keyPackage);
    let eveState = await joinChat(added.welcome!, eve.keyPackage);

    // Eve can read the group while she is a member.
    const before = await encryptMessage(aliceState, utf8("до исключения"));
    aliceState = before.state;
    const eveBefore = await expectMessage(eveState, before.message);
    eveState = eveBefore.state;
    expect(eveBefore.text).toBe("до исключения");
    const bobBefore = await expectMessage(bobState, before.message);
    bobState = bobBefore.state;

    // Alice removes eve (e.g. RBAC demotion) → new epoch.
    const removal = await removeMember(aliceState, "eve");
    aliceState = removal.state;
    expect(listMembers(aliceState).map((m) => m.userId).sort()).toEqual(["alice", "bob"]);

    // Bob applies the commit and can still read.
    const bobAfterCommit = await processIncoming(bobState, removal.commit);
    expect(bobAfterCommit.kind).toBe("handshake");
    bobState = bobAfterCommit.state;

    const after = await encryptMessage(aliceState, utf8("после исключения"));
    aliceState = after.state;
    const bobAfter = await expectMessage(bobState, after.message);
    expect(bobAfter.text).toBe("после исключения");

    // Eve's stale state must NOT decrypt the new-epoch message.
    await expect(processIncoming(eveState, after.message)).rejects.toThrow();
  });

  it("self-update (empty commit) rotates the epoch — PCS lever", async () => {
    const alice = await makeUser("alice");
    const bob = await makeUser("bob");

    let aliceState = await createChat("group-2", alice.keyPackage);
    const added = await addMembers(aliceState, [bob.keyPackage.keyPackageMessage]);
    aliceState = added.state;
    let bobState = await joinChat(added.welcome!, bob.keyPackage);

    const epochBefore = currentEpoch(aliceState);
    const update = await selfUpdate(aliceState);
    aliceState = update.state;
    expect(currentEpoch(aliceState)).toBe(epochBefore + 1n);

    const applied = await processIncoming(bobState, update.commit);
    expect(applied.kind).toBe("handshake");
    bobState = applied.state;

    const msg = await encryptMessage(bobState, utf8("новая эпоха"));
    const got = await expectMessage(aliceState, msg.message);
    expect(got.text).toBe("новая эпоха");
  });
});
