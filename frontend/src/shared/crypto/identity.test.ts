// @vitest-environment node
import { describe, expect, it } from "vitest";
import { MemoryKeyStore } from "./keystore";
import { loadOrCreateIdentity, signDetached, verifyDetached, vouchDevice, verifyVouch } from "./identity";
import { fingerprintHex, safetyNumber, formatSafetyNumber } from "./fingerprint";

describe("device identity", () => {
  it("creates once and reloads the same identity", async () => {
    const store = new MemoryKeyStore();
    const first = await loadOrCreateIdentity(store);
    const second = await loadOrCreateIdentity(store);
    expect(second.deviceId).toBe(first.deviceId);
    expect(second.publicKey).toEqual(first.publicKey);
    expect(second.seed).toEqual(first.seed);
    expect(first.publicKey).toHaveLength(32);
    expect(first.seed).toHaveLength(32);
  });

  it("signs and verifies; rejects tampered messages", async () => {
    const identity = await loadOrCreateIdentity(new MemoryKeyStore());
    const message = new TextEncoder().encode("key package bytes");
    const signature = await signDetached(identity, message);

    expect(await verifyDetached(identity.publicKey, message, signature)).toBe(true);

    const tampered = new Uint8Array(message);
    tampered[0] ^= 1;
    expect(await verifyDetached(identity.publicKey, tampered, signature)).toBe(false);
  });

  it("cross-signing vouch binds old device to new device key", async () => {
    const oldDevice = await loadOrCreateIdentity(new MemoryKeyStore());
    const newDevice = await loadOrCreateIdentity(new MemoryKeyStore());
    const attacker = await loadOrCreateIdentity(new MemoryKeyStore());

    const vouch = await vouchDevice(oldDevice, newDevice.publicKey);
    expect(await verifyVouch(oldDevice.publicKey, newDevice.publicKey, vouch)).toBe(true);
    // A vouch for one key must not validate another key (server swap attempt).
    expect(await verifyVouch(oldDevice.publicKey, attacker.publicKey, vouch)).toBe(false);
    // A vouch is not a generic signature: domain separation must hold.
    expect(await verifyDetached(oldDevice.publicKey, newDevice.publicKey, vouch)).toBe(false);
  });
});

describe("fingerprints and safety numbers", () => {
  it("fingerprint is stable and distinct per key", async () => {
    const a = await loadOrCreateIdentity(new MemoryKeyStore());
    const b = await loadOrCreateIdentity(new MemoryKeyStore());
    expect(await fingerprintHex(a.publicKey)).toBe(await fingerprintHex(a.publicKey));
    expect(await fingerprintHex(a.publicKey)).not.toBe(await fingerprintHex(b.publicKey));
  });

  it("safety number is 60 digits, symmetric, and changes when a key is swapped", async () => {
    const a = await loadOrCreateIdentity(new MemoryKeyStore());
    const b = await loadOrCreateIdentity(new MemoryKeyStore());
    const mitm = await loadOrCreateIdentity(new MemoryKeyStore());

    const alice = { userId: "user-a", deviceKeys: [a.publicKey] };
    const bob = { userId: "user-b", deviceKeys: [b.publicKey] };

    const sn = await safetyNumber(alice, bob);
    expect(sn).toMatch(/^\d{60}$/);
    expect(await safetyNumber(bob, alice)).toBe(sn);
    expect(formatSafetyNumber(sn).split(" ")).toHaveLength(12);

    // Server injects an extra "device" for bob → number must change on both ends.
    const bobWithMitm = { userId: "user-b", deviceKeys: [b.publicKey, mitm.publicKey] };
    expect(await safetyNumber(alice, bobWithMitm)).not.toBe(sn);
  });
});
