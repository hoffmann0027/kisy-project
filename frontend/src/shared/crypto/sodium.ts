// Lazy libsodium initialization. Every crypto module goes through here so the
// WASM bundle is loaded once and only when E2EE is actually used.
import _sodium from "libsodium-wrappers-sumo";

let instance: Promise<typeof _sodium> | null = null;

export type Sodium = typeof _sodium;

export function getSodium(): Promise<Sodium> {
  if (!instance) instance = _sodium.ready.then(() => _sodium);
  return instance;
}
