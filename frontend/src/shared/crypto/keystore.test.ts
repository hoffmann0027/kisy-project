// @vitest-environment node
import { beforeEach, describe, expect, it } from "vitest";
import "fake-indexeddb/auto";
import { IDBFactory } from "fake-indexeddb";
import { EncryptedIndexedDbKeyStore, MemoryKeyStore } from "./keystore";

describe("MemoryKeyStore", () => {
  it("round-trips, lists by prefix, removes", async () => {
    const store = new MemoryKeyStore();
    await store.put("mls/group/1", new Uint8Array([1, 2, 3]));
    await store.put("mls/group/2", new Uint8Array([4]));
    await store.put("device/identity", new Uint8Array([5]));

    expect(await store.get("mls/group/1")).toEqual(new Uint8Array([1, 2, 3]));
    expect(await store.list("mls/")).toEqual(["mls/group/1", "mls/group/2"]);
    await store.remove("mls/group/1");
    expect(await store.get("mls/group/1")).toBeNull();
  });
});

describe("EncryptedIndexedDbKeyStore", () => {
  beforeEach(() => {
    // Fresh fake IndexedDB per test.
    globalThis.indexedDB = new IDBFactory();
  });

  it("round-trips values across reopen", async () => {
    const store = await EncryptedIndexedDbKeyStore.open("kisy-test");
    const secret = new TextEncoder().encode("very private key material");
    await store.put("device/identity", secret);

    const reopened = await EncryptedIndexedDbKeyStore.open("kisy-test");
    expect(Array.from((await reopened.get("device/identity"))!)).toEqual(Array.from(secret));
    expect(await reopened.list("device/")).toEqual(["device/identity"]);
  });

  it("stores only ciphertext at rest (raw DB bytes ≠ plaintext)", async () => {
    const store = await EncryptedIndexedDbKeyStore.open("kisy-test");
    const plaintext = "SUPER_SECRET_SEED_0123456789";
    await store.put("device/identity", new TextEncoder().encode(plaintext));

    // Read the raw stored value directly, bypassing the store's decryption.
    const raw = await new Promise<ArrayBuffer>((resolve, reject) => {
      const open = indexedDB.open("kisy-test", 1);
      open.onsuccess = () => {
        const req = open.result.transaction("blobs", "readonly").objectStore("blobs").get("device/identity");
        req.onsuccess = () => resolve(req.result as ArrayBuffer);
        req.onerror = () => reject(req.error);
      };
      open.onerror = () => reject(open.error);
    });

    const rawText = new TextDecoder("utf-8", { fatal: false }).decode(raw);
    expect(rawText).not.toContain(plaintext);
    // AES-GCM overhead: IV (12) + tag (16).
    expect(new Uint8Array(raw).length).toBe(12 + plaintext.length + 16);
  });
});
