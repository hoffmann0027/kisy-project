import "@testing-library/jest-dom/vitest";

// jsdom ships crypto.getRandomValues but no SubtleCrypto; the E2EE core
// (ts-mls, @hpke/core, keystore encryption) needs the real WebCrypto from Node.
import { webcrypto } from "node:crypto";

if (!globalThis.crypto?.subtle) {
  Object.defineProperty(globalThis, "crypto", { value: webcrypto, configurable: true });
}
