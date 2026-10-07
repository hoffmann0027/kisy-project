// Versions of the two documents a person accepts before using KISY.
//
// The server keeps the same two values (backend/internal/consent) and refuses
// an acceptance of anything else: agreeing to last month's rules is not
// agreeing to this month's. A Go test reads THIS file and fails if the two
// sides drift apart, so change both together — and change a version whenever
// the meaning of its text changes, because that is what asks every account to
// accept again at its next sign-in.

export const PRIVACY_VERSION = "2026-10-07";
export const RULES_VERSION = "2026-10-07";

export interface Acceptance {
  privacyVersion: string;
  rulesVersion: string;
}

export function currentAcceptance(): Acceptance {
  return { privacyVersion: PRIVACY_VERSION, rulesVersion: RULES_VERSION };
}
