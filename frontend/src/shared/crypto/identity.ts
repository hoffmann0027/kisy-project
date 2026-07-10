// Device identity: an Ed25519 key pair generated on first login and never
// leaving the device (docs/e2ee-design.md §3). The same key signs MLS key
// packages (as the MLS signature key) and cross-signing vouches for newly
// linked devices (§4).
import { getSodium } from "./sodium";
import type { KeyStore } from "./keystore";

export interface DeviceIdentity {
  deviceId: string;
  /** Ed25519 public key, 32 bytes. Published to the key directory. */
  publicKey: Uint8Array;
  /** Ed25519 seed, 32 bytes. Private; also used as the MLS signature key. */
  seed: Uint8Array;
}

const IDENTITY_ENTRY = "device/identity";

// Domain separation for cross-signing vouches so a vouch signature can never
// be replayed as some other kind of signed statement.
const VOUCH_CONTEXT = new TextEncoder().encode("KISY-device-vouch-v1");

export async function loadOrCreateIdentity(store: KeyStore): Promise<DeviceIdentity> {
  const existing = await store.get(IDENTITY_ENTRY);
  if (existing) return decodeIdentity(existing);

  const sodium = await getSodium();
  const { publicKey, privateKey } = sodium.crypto_sign_keypair();
  const identity: DeviceIdentity = {
    deviceId: crypto.randomUUID(),
    publicKey,
    seed: sodium.crypto_sign_ed25519_sk_to_seed(privateKey),
  };
  await store.put(IDENTITY_ENTRY, encodeIdentity(identity));
  return identity;
}

function encodeIdentity(identity: DeviceIdentity): Uint8Array {
  const deviceId = new TextEncoder().encode(identity.deviceId);
  const out = new Uint8Array(1 + deviceId.length + 32 + 32);
  out[0] = deviceId.length;
  out.set(deviceId, 1);
  out.set(identity.publicKey, 1 + deviceId.length);
  out.set(identity.seed, 1 + deviceId.length + 32);
  return out;
}

function decodeIdentity(bytes: Uint8Array): DeviceIdentity {
  const idLen = bytes[0];
  return {
    deviceId: new TextDecoder().decode(bytes.slice(1, 1 + idLen)),
    publicKey: bytes.slice(1 + idLen, 1 + idLen + 32),
    seed: bytes.slice(1 + idLen + 32, 1 + idLen + 64),
  };
}

async function fullPrivateKey(seed: Uint8Array): Promise<Uint8Array> {
  const sodium = await getSodium();
  return sodium.crypto_sign_seed_keypair(seed).privateKey;
}

export async function signDetached(identity: DeviceIdentity, message: Uint8Array): Promise<Uint8Array> {
  const sodium = await getSodium();
  return sodium.crypto_sign_detached(message, await fullPrivateKey(identity.seed));
}

export async function verifyDetached(
  publicKey: Uint8Array,
  message: Uint8Array,
  signature: Uint8Array,
): Promise<boolean> {
  const sodium = await getSodium();
  return sodium.crypto_sign_verify_detached(signature, message, publicKey);
}

function vouchMessage(newDevicePublicKey: Uint8Array): Uint8Array {
  const msg = new Uint8Array(VOUCH_CONTEXT.length + newDevicePublicKey.length);
  msg.set(VOUCH_CONTEXT);
  msg.set(newDevicePublicKey, VOUCH_CONTEXT.length);
  return msg;
}

/** An existing device signs a newly linked device's identity key (QR flow, §4). */
export async function vouchDevice(
  existing: DeviceIdentity,
  newDevicePublicKey: Uint8Array,
): Promise<Uint8Array> {
  return signDetached(existing, vouchMessage(newDevicePublicKey));
}

export async function verifyVouch(
  existingDevicePublicKey: Uint8Array,
  newDevicePublicKey: Uint8Array,
  signature: Uint8Array,
): Promise<boolean> {
  return verifyDetached(existingDevicePublicKey, vouchMessage(newDevicePublicKey), signature);
}
