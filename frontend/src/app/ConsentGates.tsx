import { useEffect, useRef, useState, type ReactNode } from "react";
import { usersApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { clearLocalConsent, hasLocalConsent, saveLocalConsent } from "@shared/lib/consent";
import { Spinner, toast } from "@shared/ui";
import { ConsentScreen } from "@pages/legal/ConsentScreen";

/**
 * Before sign-in and sign-up: nobody reaches either form on this device
 * without ticking both boxes. The tick is kept on the device until sign-out
 * (shared/lib/consent.ts) and travels to the server with the sign-up, or
 * right after the sign-in (AccountConsentGate).
 */
export function RequireLocalConsent({ children }: { children: ReactNode }) {
  const [accepted, setAccepted] = useState(hasLocalConsent);
  if (!accepted) {
    return (
      <ConsentScreen
        onAccept={() => {
          saveLocalConsent();
          setAccepted(true);
        }}
      />
    );
  }
  return <>{children}</>;
}

/**
 * After sign-in: an account that has not accepted the current versions stays
 * here. Two ways through:
 *
 * - the person ticked both boxes on this device just before signing in —
 *   that tick is theirs (it is cleared at every sign-out), so it is recorded
 *   for the account without asking a second time;
 * - otherwise (signed in before this screen existed, or the texts changed
 *   since) the same two boxes are shown here.
 */
export function AccountConsentGate() {
  const setUser = useAuthStore((s) => s.setUser);
  const [busy, setBusy] = useState(false);
  const [ask, setAsk] = useState(!hasLocalConsent());
  const tried = useRef(false);

  const submit = async (fromScreen: boolean) => {
    setBusy(true);
    try {
      const { user } = await usersApi.acceptConsent();
      saveLocalConsent();
      setUser(user);
    } catch {
      // A stale tick (the texts changed after it) or a failed request: ask in
      // person rather than loop.
      clearLocalConsent();
      setAsk(true);
      if (fromScreen) toast.error("Не удалось сохранить согласие. Попробуйте ещё раз");
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    if (ask || tried.current) return;
    tried.current = true;
    void submit(false);
  }, []); // once, on mount: the ref guards against a second run in StrictMode

  if (!ask) {
    // The silent submit is in flight.
    return (
      <div style={{ height: "100%", display: "flex", alignItems: "center", justifyContent: "center" }}>
        <Spinner size={32} />
      </div>
    );
  }
  return <ConsentScreen reason="account" busy={busy} onAccept={() => submit(true)} />;
}
