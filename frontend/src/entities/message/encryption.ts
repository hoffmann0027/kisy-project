// Fail-closed encryption for private chats (audit A-10).
//
// A private chat carries text only as end-to-end ciphertext. Every way that
// could go wrong — no session on this device, an unknown peer, a peer with no
// key packages, an MLS or network error mid-handshake — ends here in a
// UserFacingError, and the caller does not send. There is no plaintext
// fallback, on purpose: a message the user believes is private must never
// reach the server readable. The server enforces the same rule (422
// E2EE_REQUIRED), so an old or modified client cannot bypass it either.
import { UserFacingError } from "@shared/lib/errors";
import { useAuthStore } from "@shared/store/auth";
import { e2eeSession, encryptForChat, initE2EE, type E2EESession, type EncryptedBody } from "@entities/e2ee";

export const ENCRYPTION_UNAVAILABLE =
  "Шифрование на этом устройстве не запустилось — сообщение не отправлено. Перезапустите приложение и повторите.";
export const ENCRYPTION_PEER_UNKNOWN = "Не удалось определить собеседника — сообщение не отправлено.";
export const ENCRYPTION_FAILED = "Не удалось зашифровать сообщение — оно не отправлено. Повторите попытку.";

/** The running E2EE session, starting it if it is not up yet; never null. */
export async function requireE2EESession(): Promise<E2EESession> {
  const running = e2eeSession();
  if (running) return running;
  const userId = useAuthStore.getState().user?.id;
  const started = userId ? await initE2EE(userId).catch(() => null) : null;
  if (!started) throw new UserFacingError(ENCRYPTION_UNAVAILABLE);
  return started;
}

/**
 * Encrypt text for a private chat or throw a UserFacingError. Never returns
 * without ciphertext.
 */
export async function encryptPrivateText(
  chatId: string,
  peerUserId: string | undefined,
  text: string,
): Promise<{ session: E2EESession; body: EncryptedBody }> {
  const session = await requireE2EESession();
  if (!peerUserId) throw new UserFacingError(ENCRYPTION_PEER_UNKNOWN);
  let body: EncryptedBody | null;
  try {
    body = await encryptForChat(session, chatId, peerUserId, text);
  } catch (err) {
    if (err instanceof UserFacingError) throw err;
    throw new UserFacingError(ENCRYPTION_FAILED, { cause: err });
  }
  if (!body) throw new UserFacingError(ENCRYPTION_FAILED);
  return { session, body };
}
