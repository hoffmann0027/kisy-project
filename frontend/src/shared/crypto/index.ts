// KISY E2EE crypto core (docs/e2ee-design.md). Runtime-independent: no React,
// no shared/api imports — usable from the web app, the Tauri webview and tests.
export { getSodium } from "./sodium";
export {
  type KeyStore,
  MemoryKeyStore,
  EncryptedIndexedDbKeyStore,
} from "./keystore";
export {
  type DeviceIdentity,
  loadOrCreateIdentity,
  signDetached,
  verifyDetached,
  vouchDevice,
  verifyVouch,
} from "./identity";
export {
  type UserKeySet,
  fingerprint,
  fingerprintHex,
  safetyNumber,
  formatSafetyNumber,
} from "./fingerprint";
export {
  KISY_CIPHERSUITE,
  KISY_E2EE_ALG,
  type ChatState,
  type DeviceKeyPackage,
  type MemberInfo,
  type CommitResult,
  type EncryptedMessage,
  type Incoming,
  createDeviceKeyPackage,
  decodeKeyPackageMessage,
  parseCredentialIdentity,
  createChat,
  joinChat,
  addMembers,
  removeMember,
  selfUpdate,
  encryptMessage,
  processIncoming,
  listMembers,
  currentEpoch,
  serializeChatState,
  deserializeChatState,
} from "./mls";
