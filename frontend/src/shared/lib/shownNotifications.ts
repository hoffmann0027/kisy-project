// Pushes the browser has drawn stay on screen until something closes them. An
// announcement taken back must take its push with it (backend
// announcements.Revoke). The server cannot do that for a browser — a web push
// that shows nothing makes the browser show a notice of its own instead — so
// the open app does it. Phones are handled natively (CallPushService.java).

const PREFIX = "announcement-";

/** The tag an announcement's push is shown under (backend announcements.PushTag). */
export function announcementPushTag(id: string): string {
  return PREFIX + id;
}

async function shownPushes(): Promise<Notification[]> {
  if (typeof navigator === "undefined" || !("serviceWorker" in navigator)) return [];
  try {
    const reg = await navigator.serviceWorker.getRegistration();
    return reg ? await reg.getNotifications() : [];
  } catch {
    // No worker, or a browser without getNotifications: nothing to close.
    return [];
  }
}

/** Close the push of one announcement, if the browser still shows it. */
export async function closeAnnouncementPush(id: string): Promise<void> {
  const tag = announcementPushTag(id);
  for (const n of await shownPushes()) if (n.tag === tag) n.close();
}

/**
 * Close the pushes of announcements no longer in the list: taken back while
 * the app was closed, so it never heard the revocation itself.
 */
export async function closeWithdrawnAnnouncementPushes(current: Iterable<string>): Promise<void> {
  const keep = new Set([...current].map(announcementPushTag));
  for (const n of await shownPushes()) {
    if (n.tag.startsWith(PREFIX) && !keep.has(n.tag)) n.close();
  }
}
