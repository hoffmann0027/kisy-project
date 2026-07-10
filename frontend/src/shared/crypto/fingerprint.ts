// Key fingerprints and safety numbers (docs/e2ee-design.md §3.3).
//
// A safety number is a 60-digit string both chat partners can compare out of
// band (read aloud, QR at a meeting). It commits to BOTH users' full device
// key sets, so a server-injected key (MITM) changes the number on both ends.
import { getSodium } from "./sodium";

const FINGERPRINT_BYTES = 32;

export async function fingerprint(publicKey: Uint8Array): Promise<Uint8Array> {
  const sodium = await getSodium();
  return sodium.crypto_generichash(FINGERPRINT_BYTES, publicKey, null);
}

/** Human-readable fingerprint: hex in groups of 4, e.g. "ab12 cd34 ...". */
export async function fingerprintHex(publicKey: Uint8Array): Promise<string> {
  const sodium = await getSodium();
  const hex = sodium.to_hex(await fingerprint(publicKey));
  return hex.match(/.{4}/g)!.join(" ");
}

export interface UserKeySet {
  userId: string;
  /** Ed25519 identity public keys of every device of this user. */
  deviceKeys: Uint8Array[];
}

// 30 digits per side: 6 groups of 5, each group derived Signal-style from
// 4 hash bytes mod 100000.
function digitsFromHash(hash: Uint8Array): string {
  let out = "";
  for (let group = 0; group < 6; group++) {
    const o = group * 4;
    const n = ((hash[o] << 24) | (hash[o + 1] << 16) | (hash[o + 2] << 8) | hash[o + 3]) >>> 0;
    out += String(n % 100000).padStart(5, "0");
  }
  return out;
}

async function userDigest(user: UserKeySet): Promise<Uint8Array> {
  const sodium = await getSodium();
  const fps = await Promise.all(user.deviceKeys.map((k) => fingerprint(k)));
  const sortedHex = fps.map((f) => sodium.to_hex(f)).sort();
  const material = new TextEncoder().encode(`KISY-safety-v1|${user.userId}|${sortedHex.join("|")}`);
  return sodium.crypto_generichash(FINGERPRINT_BYTES, material, null);
}

/**
 * 60-digit safety number for a pair of users. Symmetric:
 * safetyNumber(a, b) === safetyNumber(b, a).
 */
export async function safetyNumber(a: UserKeySet, b: UserKeySet): Promise<string> {
  const sodium = await getSodium();
  const [da, db] = await Promise.all([userDigest(a), userDigest(b)]);
  const halves = [digitsFromHash(da), digitsFromHash(db)];
  // Order by digest so both sides render the same string.
  if (sodium.to_hex(da) > sodium.to_hex(db)) halves.reverse();
  return halves.join("");
}

/** Format a safety number for display: groups of 5 digits. */
export function formatSafetyNumber(sn: string): string {
  return sn.match(/.{5}/g)!.join(" ");
}
