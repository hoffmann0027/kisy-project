// MLS (RFC 9420) wrapper around ts-mls — the single group protocol for both
// 1:1 chats (a 2-member group) and group chats (docs/e2ee-design.md §1).
//
// This module is the ONLY place that touches ts-mls types; the rest of the
// app sees opaque `ChatState` and byte arrays, so swapping the MLS library
// later stays a local change.
import {
  createApplicationMessage,
  createCommit,
  createGroup,
  joinGroup,
  processMessage,
  acceptAll,
  getCiphersuiteImpl,
  getCiphersuiteFromName,
  generateKeyPackageWithKey,
  defaultCapabilities,
  defaultLifetime,
  emptyPskIndex,
  encodeMlsMessage,
  decodeMlsMessage,
  encodeGroupState,
  decodeGroupState,
  zeroOutUint8Array,
  type ClientState,
  type CiphersuiteImpl,
  type Credential,
  type KeyPackage,
  type PrivateKeyPackage,
  type Proposal,
} from "ts-mls";
import { defaultClientConfig } from "ts-mls/clientConfig.js";
import type { DeviceIdentity } from "./identity";

/** Pinned ciphersuite: X25519 + ChaCha20-Poly1305 + Ed25519 (design §1). */
export const KISY_CIPHERSUITE = "MLS_128_DHKEMX25519_CHACHA20POLY1305_SHA256_Ed25519";

/** Version tag stored alongside ciphertext (messages.alg) for crypto migrations. */
export const KISY_E2EE_ALG = 1;

export type ChatState = ClientState;

export interface DeviceKeyPackage {
  /** Encoded MLSMessage(key package) — uploaded to the server directory. */
  keyPackageMessage: Uint8Array;
  publicPackage: KeyPackage;
  /** Stays on the device; needed to join groups via Welcome. */
  privatePackage: PrivateKeyPackage;
}

export interface MemberInfo {
  userId: string;
  deviceId: string;
  leafIndex: number;
  signaturePublicKey: Uint8Array;
}

let implPromise: Promise<CiphersuiteImpl> | null = null;

export function getMlsImpl(): Promise<CiphersuiteImpl> {
  if (!implPromise) implPromise = getCiphersuiteImpl(getCiphersuiteFromName(KISY_CIPHERSUITE));
  return implPromise;
}

// Credential identity: "userId/deviceId". userId is a UUID (no "/" inside),
// so the first "/" is an unambiguous separator.
function credentialFor(userId: string, deviceId: string): Credential {
  return { credentialType: "basic", identity: new TextEncoder().encode(`${userId}/${deviceId}`) };
}

export function parseCredentialIdentity(identity: Uint8Array): { userId: string; deviceId: string } | null {
  const text = new TextDecoder().decode(identity);
  const sep = text.indexOf("/");
  if (sep <= 0) return null;
  return { userId: text.slice(0, sep), deviceId: text.slice(sep + 1) };
}

/**
 * Generate a one-time MLS key package bound to the device identity key:
 * the Ed25519 seed doubles as the MLS signature key, so everything the
 * device publishes is verifiable against one identity (design §3.1).
 */
export async function createDeviceKeyPackage(
  identity: DeviceIdentity,
  userId: string,
): Promise<DeviceKeyPackage> {
  const impl = await getMlsImpl();
  const { publicPackage, privatePackage } = await generateKeyPackageWithKey(
    credentialFor(userId, identity.deviceId),
    defaultCapabilities(),
    defaultLifetime,
    [],
    { signKey: identity.seed, publicKey: identity.publicKey },
    impl,
  );
  const keyPackageMessage = encodeMlsMessage({
    keyPackage: publicPackage,
    wireformat: "mls_key_package",
    version: "mls10",
  });
  return { keyPackageMessage, publicPackage, privatePackage };
}

export function decodeKeyPackageMessage(bytes: Uint8Array): KeyPackage {
  const decoded = decodeMlsMessage(bytes, 0)?.[0];
  if (!decoded || decoded.wireformat !== "mls_key_package") {
    throw new Error("Not an MLS key package");
  }
  return decoded.keyPackage;
}

/** Create a new chat group with only ourselves in it. */
export async function createChat(chatId: string, ownPackage: DeviceKeyPackage): Promise<ChatState> {
  const impl = await getMlsImpl();
  return createGroup(
    new TextEncoder().encode(chatId),
    ownPackage.publicPackage,
    ownPackage.privatePackage,
    [],
    impl,
  );
}

export interface CommitResult {
  state: ChatState;
  /** Encoded commit MLSMessage — fan out to current members. */
  commit: Uint8Array;
  /** Encoded welcome MLSMessage — deliver to just-added devices. */
  welcome: Uint8Array | null;
  epoch: bigint;
}

async function commitProposals(state: ChatState, proposals: Proposal[]): Promise<CommitResult> {
  const impl = await getMlsImpl();
  const result = await createCommit(
    { state, cipherSuite: impl },
    // ratchetTreeExtension: joiners get the tree inside the Welcome, so the
    // server never needs to serve (or be trusted for) the ratchet tree.
    { extraProposals: proposals, ratchetTreeExtension: true },
  );
  result.consumed.forEach(zeroOutUint8Array);
  return {
    state: result.newState,
    commit: encodeMlsMessage(result.commit),
    welcome: result.welcome
      ? encodeMlsMessage({ welcome: result.welcome, wireformat: "mls_welcome", version: "mls10" })
      : null,
    epoch: result.newState.groupContext.epoch,
  };
}

/** Add devices by their (server-served, signature-verified) key packages. */
export async function addMembers(state: ChatState, keyPackageMessages: Uint8Array[]): Promise<CommitResult> {
  const proposals: Proposal[] = keyPackageMessages.map((bytes) => ({
    proposalType: "add",
    add: { keyPackage: decodeKeyPackageMessage(bytes) },
  }));
  return commitProposals(state, proposals);
}

/**
 * Remove all devices of `userId` (or one device if `deviceId` given).
 * The commit starts a new epoch: the removed member cannot decrypt anything
 * sent after it — this is the RBAC hook of design §5.1.
 */
export async function removeMember(state: ChatState, userId: string, deviceId?: string): Promise<CommitResult> {
  const targets = listMembers(state).filter(
    (m) => m.userId === userId && (deviceId === undefined || m.deviceId === deviceId),
  );
  if (targets.length === 0) throw new Error("Member not found in group");
  const proposals: Proposal[] = targets.map((m) => ({
    proposalType: "remove",
    remove: { removed: m.leafIndex },
  }));
  return commitProposals(state, proposals);
}

/**
 * Empty commit: rotates our own leaf keys and the group secret — the MLS
 * post-compromise-security lever. Run periodically (design §8.4).
 */
export async function selfUpdate(state: ChatState): Promise<CommitResult> {
  return commitProposals(state, []);
}

/** Join a chat from a Welcome message addressed to our key package. */
export async function joinChat(welcomeMessage: Uint8Array, ownPackage: DeviceKeyPackage): Promise<ChatState> {
  const impl = await getMlsImpl();
  const decoded = decodeMlsMessage(welcomeMessage, 0)?.[0];
  if (!decoded || decoded.wireformat !== "mls_welcome") throw new Error("Not an MLS welcome");
  return joinGroup(decoded.welcome, ownPackage.publicPackage, ownPackage.privatePackage, emptyPskIndex, impl);
}

export interface EncryptedMessage {
  state: ChatState;
  /** Encoded private MLSMessage — this is what the server stores. */
  message: Uint8Array;
  epoch: bigint;
}

export async function encryptMessage(state: ChatState, plaintext: Uint8Array): Promise<EncryptedMessage> {
  const impl = await getMlsImpl();
  const result = await createApplicationMessage(state, plaintext, impl);
  result.consumed.forEach(zeroOutUint8Array);
  return {
    state: result.newState,
    message: encodeMlsMessage({
      privateMessage: result.privateMessage,
      wireformat: "mls_private_message",
      version: "mls10",
    }),
    epoch: result.newState.groupContext.epoch,
  };
}

export type Incoming =
  | { kind: "message"; state: ChatState; plaintext: Uint8Array }
  | { kind: "handshake"; state: ChatState };

/**
 * Process an incoming MLS message: an application message yields plaintext,
 * a handshake message (commit/proposal — e.g. we were in a group where
 * someone was added/removed) advances the group state.
 */
export async function processIncoming(state: ChatState, messageBytes: Uint8Array): Promise<Incoming> {
  const impl = await getMlsImpl();
  const decoded = decodeMlsMessage(messageBytes, 0)?.[0];
  if (!decoded) throw new Error("Cannot decode MLS message");
  if (decoded.wireformat !== "mls_private_message" && decoded.wireformat !== "mls_public_message") {
    throw new Error(`Unexpected MLS wireformat: ${decoded.wireformat}`);
  }
  const result = await processMessage(decoded, state, emptyPskIndex, acceptAll, impl);
  if (result.kind === "applicationMessage") {
    result.consumed.forEach(zeroOutUint8Array);
    return { kind: "message", state: result.newState, plaintext: result.message };
  }
  result.consumed.forEach(zeroOutUint8Array);
  return { kind: "handshake", state: result.newState };
}

/** Current members from the ratchet tree (leaves sit at even node indices). */
export function listMembers(state: ChatState): MemberInfo[] {
  const members: MemberInfo[] = [];
  state.ratchetTree.forEach((node, nodeIndex) => {
    if (nodeIndex % 2 !== 0 || node === undefined || node.nodeType !== "leaf") return;
    const credential = node.leaf.credential;
    if (credential.credentialType !== "basic") return;
    const parsed = parseCredentialIdentity(credential.identity);
    if (!parsed) return;
    members.push({
      ...parsed,
      leafIndex: nodeIndex / 2,
      signaturePublicKey: node.leaf.signaturePublicKey,
    });
  });
  return members;
}

export function currentEpoch(state: ChatState): bigint {
  return state.groupContext.epoch;
}

// Group state persistence: serialized bytes go into the KeyStore, which
// encrypts them at rest with the device-local key.
export function serializeChatState(state: ChatState): Uint8Array {
  return encodeGroupState(state);
}

export function deserializeChatState(bytes: Uint8Array): ChatState {
  const decoded = decodeGroupState(bytes, 0);
  if (!decoded) throw new Error("Cannot decode MLS group state");
  return { ...decoded[0], clientConfig: defaultClientConfig };
}
