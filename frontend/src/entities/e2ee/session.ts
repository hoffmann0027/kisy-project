// E2EE device session: bootstrapped once after login (docs/e2ee-design.md §3).
// Owns the per-user encrypted keystore, the device identity and the pool of
// one-time key packages (public halves uploaded to the directory, private
// halves kept here — they are needed to join groups via Welcome).
import {
  EncryptedIndexedDbKeyStore,
  loadOrCreateIdentity,
  createDeviceKeyPackage,
  getSodium,
  type DeviceIdentity,
  type DeviceKeyPackage,
  type KeyStore,
} from "@shared/crypto";
import { e2eeApi } from "@shared/api/endpoints";

export interface E2EESession {
  store: KeyStore;
  identity: DeviceIdentity;
  userId: string;
}

// Replenish the server-side pool when it drops below MIN, up to TARGET.
const POOL_MIN = 10;
const POOL_TARGET = 30;
const POOL_INDEX = "kp/index";

let session: E2EESession | null = null;
let initPromise: Promise<E2EESession | null> | null = null;

export function e2eeSession(): E2EESession | null {
  return session;
}

/**
 * Initialize E2EE for the logged-in user: open the encrypted keystore,
 * create/load the device identity, announce it to the directory and top up
 * the one-time key package pool. Failure disables E2EE for the session
 * (messages fall back to plaintext) rather than breaking messaging.
 */
export function initE2EE(userId: string): Promise<E2EESession | null> {
  if (session && session.userId === userId) return Promise.resolve(session);
  if (!initPromise) {
    initPromise = bootstrap(userId).catch((err) => {
      console.warn("E2EE init failed; falling back to plaintext", err);
      initPromise = null;
      return null;
    });
  }
  return initPromise;
}

/** Test-only: reset module state between tests. */
export function resetE2EEForTests(): void {
  session = null;
  initPromise = null;
}

async function bootstrap(userId: string): Promise<E2EESession> {
  const store = await EncryptedIndexedDbKeyStore.open(`kisy-e2ee-${userId}`);
  const identity = await loadOrCreateIdentity(store);
  const sodium = await getSodium();

  await e2eeApi.registerDevice({
    deviceId: identity.deviceId,
    name: deviceName(),
    ed25519Pub: sodium.to_base64(identity.publicKey, sodium.base64_variants.ORIGINAL),
  });

  session = { store, identity, userId };
  await replenishKeyPackages(session);
  return session;
}

function deviceName(): string {
  const ua = navigator.userAgent;
  if (/mobile/i.test(ua)) return "Мобильный браузер";
  return "Браузер";
}

// --- key package pool ---
//
// Each generated key package is stored locally as kp/<n>: the encoded public
// message plus the three private keys. When a Welcome arrives we try pool
// entries until one opens it (welcomes address a specific key package).

interface StoredKeyPackage {
  message: string; // base64 encoded MLS key package message
  init: string;
  hpke: string;
  sig: string;
}

async function replenishKeyPackages(s: E2EESession): Promise<void> {
  const { available } = await e2eeApi.countKeyPackages(s.identity.deviceId);
  if (available >= POOL_MIN) return;
  const fresh = await topUpKeyPackages(s, POOL_TARGET - available);
  await e2eeApi.uploadKeyPackages(s.identity.deviceId, fresh);
}

/**
 * Generate `count` one-time key packages, keep their private halves locally
 * and return the encoded public messages (base64) for upload.
 */
export async function topUpKeyPackages(s: E2EESession, count: number): Promise<string[]> {
  const sodium = await getSodium();
  const b64 = (u: Uint8Array) => sodium.to_base64(u, sodium.base64_variants.ORIGINAL);

  const index = await readPoolIndex(s.store);
  const fresh: string[] = [];
  for (let i = 0; i < count; i++) {
    const kp = await createDeviceKeyPackage(s.identity, s.userId);
    const n = index.next++;
    const stored: StoredKeyPackage = {
      message: b64(kp.keyPackageMessage),
      init: b64(kp.privatePackage.initPrivateKey),
      hpke: b64(kp.privatePackage.hpkePrivateKey),
      sig: b64(kp.privatePackage.signaturePrivateKey),
    };
    await s.store.put(`kp/${n}`, new TextEncoder().encode(JSON.stringify(stored)));
    index.entries.push(n);
    fresh.push(stored.message);
  }
  await writePoolIndex(s.store, index);
  return fresh;
}

interface PoolIndex {
  next: number;
  entries: number[];
}

async function readPoolIndex(store: KeyStore): Promise<PoolIndex> {
  const raw = await store.get(POOL_INDEX);
  if (!raw) return { next: 0, entries: [] };
  return JSON.parse(new TextDecoder().decode(raw)) as PoolIndex;
}

async function writePoolIndex(store: KeyStore, index: PoolIndex): Promise<void> {
  await store.put(POOL_INDEX, new TextEncoder().encode(JSON.stringify(index)));
}

/** All locally stored key packages (for opening Welcomes). */
export async function localKeyPackages(s: E2EESession): Promise<{ n: number; pkg: DeviceKeyPackage }[]> {
  const sodium = await getSodium();
  const fromB64 = (t: string) => sodium.from_base64(t, sodium.base64_variants.ORIGINAL);
  const { decodeKeyPackageMessage } = await import("@shared/crypto");

  const index = await readPoolIndex(s.store);
  const out: { n: number; pkg: DeviceKeyPackage }[] = [];
  for (const n of index.entries) {
    const raw = await s.store.get(`kp/${n}`);
    if (!raw) continue;
    const stored = JSON.parse(new TextDecoder().decode(raw)) as StoredKeyPackage;
    const keyPackageMessage = fromB64(stored.message);
    out.push({
      n,
      pkg: {
        keyPackageMessage,
        publicPackage: decodeKeyPackageMessage(keyPackageMessage),
        privatePackage: {
          initPrivateKey: fromB64(stored.init),
          hpkePrivateKey: fromB64(stored.hpke),
          signaturePrivateKey: fromB64(stored.sig),
        },
      },
    });
  }
  return out;
}

/** Drop a consumed pool entry (its Welcome was processed). */
export async function dropKeyPackage(s: E2EESession, n: number): Promise<void> {
  await s.store.remove(`kp/${n}`);
  const index = await readPoolIndex(s.store);
  index.entries = index.entries.filter((e) => e !== n);
  await writePoolIndex(s.store, index);
}
