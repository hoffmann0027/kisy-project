// @vitest-environment node
//
// Audit A-11: initE2EE handed back whatever session start was in flight, for
// whichever user asked. The second account signed in on a page got the first
// one's device identity and keystore — and encrypted as them.
import { beforeEach, describe, expect, it, vi } from "vitest";

const crypto = vi.hoisted(() => ({
  opened: [] as string[],
  release: null as null | (() => void),
}));

vi.mock("@shared/crypto", () => ({
  EncryptedIndexedDbKeyStore: {
    open: async (name: string) => {
      crypto.opened.push(name);
      if (crypto.release === null) {
        await new Promise<void>((r) => (crypto.release = r));
      }
      return { get: async () => null, put: async () => {}, remove: async () => {}, list: async () => [] };
    },
  },
  loadOrCreateIdentity: async () => ({ deviceId: "dev-1", publicKey: new Uint8Array(32) }),
  createDeviceKeyPackage: vi.fn(),
  getSodium: async () => ({ to_base64: () => "", base64_variants: { ORIGINAL: 1 } }),
  requestPersistentStorage: async () => true,
}));

vi.mock("@shared/api/endpoints", () => ({
  e2eeApi: {
    registerDevice: async () => ({}),
    countKeyPackages: async () => ({ available: 30 }),
    uploadKeyPackages: async () => ({}),
  },
}));

const { initE2EE, e2eeSession, resetE2EEForTests } = await import("./session");

beforeEach(() => {
  resetE2EEForTests();
  crypto.opened = [];
  crypto.release = null;
});

describe("one account's E2EE session never serves another", () => {
  it("does not hand a started session to a different user", async () => {
    crypto.release = () => {};
    const alice = await initE2EE("alice");
    expect(alice?.userId).toBe("alice");

    const forBob = await initE2EE("bob");
    expect(forBob).toBeNull();
    expect(e2eeSession()?.userId).toBe("alice");
  });

  it("does not hand a start still in flight to a different user", async () => {
    const aliceStarting = initE2EE("alice");
    const forBob = await initE2EE("bob");
    expect(forBob).toBeNull();

    crypto.release?.();
    expect((await aliceStarting)?.userId).toBe("alice");
    expect(crypto.opened).toEqual(["kisy-e2ee-alice"]);
  });
});
