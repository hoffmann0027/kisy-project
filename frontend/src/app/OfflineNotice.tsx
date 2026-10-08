import { useEffect, useState } from "react";
import { Button } from "@shared/ui";
import { useAuthStore } from "@shared/store/auth";
import { t } from "@shared/i18n";

// Shown when the app cannot reach the server on start.
//
// This is the screen that used to be the password form. Being unable to ask
// the server who you are is not the same as being signed out, and telling
// someone to log in again — when their session was never in question — is both
// wrong and, on a phone, genuinely hard to do: they are usually somewhere
// without signal, being asked for a long password they may not remember.

export function OfflineNotice() {
  const bootstrap = useAuthStore((s) => s.bootstrap);
  const [retrying, setRetrying] = useState(false);

  const retry = () => {
    setRetrying(true);
    void bootstrap().finally(() => setRetrying(false));
  };

  useEffect(() => {
    // The usual ending: the radio attaches a second later and nobody has to
    // press anything.
    const onOnline = () => void bootstrap();
    const onVisible = () => {
      if (document.visibilityState === "visible") void bootstrap();
    };
    window.addEventListener("online", onOnline);
    document.addEventListener("visibilitychange", onVisible);
    return () => {
      window.removeEventListener("online", onOnline);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [bootstrap]);

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 14,
        padding: 24,
        textAlign: "center",
      }}
    >
      <div style={{ fontSize: 17, fontWeight: 640 }}>{t("common.offline.title")}</div>
      <div style={{ color: "var(--color-text-secondary)", fontSize: 14, maxWidth: 320 }}>
        {t("common.offline.body")}
      </div>
      <Button variant="secondary" onClick={retry} loading={retrying}>
        {t("common.offline.retry")}
      </Button>
    </div>
  );
}
