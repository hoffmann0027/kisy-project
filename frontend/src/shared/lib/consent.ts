import { currentAcceptance, type Acceptance } from "@shared/config/legalVersions";

// The tick on the first screen, before anyone has signed in.
//
// The record that counts lives on the server (backend/internal/consent):
// sign-up sends it with the account, and an existing account accepts through
// /users/me/consent. What is kept HERE is only "the person in front of this
// device ticked both boxes", so the sign-in and sign-up screens open, and so
// the server record can be written right after sign-in without asking twice.
//
// It is cleared on sign-out. Otherwise the next person to sign in on a shared
// phone would inherit the previous one's tick, and their account would be
// recorded as having agreed to something they never saw.

const KEY = "kisy.consent";

export function hasLocalConsent(): boolean {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return false;
    const saved = JSON.parse(raw) as Partial<Acceptance>;
    const now = currentAcceptance();
    // A tick on an older text is not a tick on this one.
    return saved.privacyVersion === now.privacyVersion && saved.rulesVersion === now.rulesVersion;
  } catch {
    return false;
  }
}

export function saveLocalConsent(): void {
  try {
    localStorage.setItem(KEY, JSON.stringify(currentAcceptance()));
  } catch {
    // A full or disabled store only means the screen shows again next time.
  }
}

export function clearLocalConsent(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    // Nothing to clear.
  }
}
