import { useCallback, useEffect, useState } from "react";
import { enablePush } from "@shared/lib/push";
import { currentPlatform, openPermissionSettings, readPermissions, requestPermission } from "./platform";
import { actionFor, type AppPermission, type PermissionState } from "./sequence";

/**
 * Current permission states, kept fresh across a trip to Settings: the page
 * that sent someone there has to notice what they changed when they come back.
 */
export function usePermissionStates(enabled = true) {
  const [states, setStates] = useState<Partial<Record<AppPermission, PermissionState>> | null>(null);
  const platform = currentPlatform();

  const refresh = useCallback(async () => {
    try {
      const next = await readPermissions();
      setStates(next);
      return next;
    } catch (err) {
      console.warn("[permissions] check failed", err);
      return null;
    }
  }, []);

  useEffect(() => {
    if (!enabled) return;
    void refresh();
    const onBack = () => {
      if (document.visibilityState === "visible") void refresh();
    };
    document.addEventListener("visibilitychange", onBack);
    window.addEventListener("focus", onBack);
    return () => {
      document.removeEventListener("visibilitychange", onBack);
      window.removeEventListener("focus", onBack);
    };
  }, [enabled, refresh]);

  /**
   * The main button: the system dialog, or Settings when there is no dialog
   * left to show. Resolves with the state afterwards.
   */
  const act = useCallback(
    async (permission: AppPermission): Promise<PermissionState | undefined> => {
      const current = states?.[permission];
      if (!current) return undefined;
      const action = actionFor(permission, current, platform);
      if (action === "settings") {
        await openPermissionSettings(permission);
        return current; // the answer arrives when the app is back on screen
      }
      if (action !== "request") return current;

      const answer = await requestPermission(permission);
      setStates((s) => ({ ...s, [permission]: answer }));
      // A notification permission on its own delivers nothing: the device
      // still has to be registered for push, which is what the profile's push
      // switch does. Doing it here spares a second step.
      if (permission === "notifications" && answer === "granted") {
        await enablePush().catch((err) => console.warn("[permissions] push registration after grant failed", err));
      }
      return answer;
    },
    [platform, states],
  );

  return { states, platform, refresh, act };
}
