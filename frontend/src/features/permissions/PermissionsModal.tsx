import { useState } from "react";
import { Button, Modal, Spinner } from "@shared/ui";
import { PERMISSION_COPY, stateLabel } from "./copy";
import { actionFor, permissionsFor, type AppPermission } from "./sequence";
import { usePermissionStates } from "./usePermissionStates";
import "./permissions.css";

// "Разрешения" in the profile: the onboarding's second chance. Every
// permission with its current state and the one thing that would fix it.

export function PermissionsModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { states, platform, act } = usePermissionStates(open);
  const [busy, setBusy] = useState<AppPermission | null>(null);

  const run = async (permission: AppPermission) => {
    setBusy(permission);
    try {
      await act(permission);
    } finally {
      setBusy(null);
    }
  };

  return (
    <Modal open={open} title="Разрешения" onClose={onClose}>
      {!states ? (
        <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
          <Spinner />
        </div>
      ) : (
        <ul className="perm-list">
          {permissionsFor(platform).map((permission) => {
            const state = states[permission];
            const copy = PERMISSION_COPY[permission];
            const action = state ? actionFor(permission, state, platform) : "done";
            const badge = state === "granted" ? " perm-badge--ok" : state === "denied" ? " perm-badge--no" : "";
            return (
              <li key={permission} className="perm-row">
                <span className="perm-row__icon">{copy.icon}</span>
                <div className="perm-row__body">
                  <div className="perm-row__head">
                    <span className="perm-row__title">{copy.title}</span>
                    <span className={"perm-badge" + badge}>{stateLabel(state)}</span>
                  </div>
                  <p className="perm-row__text">
                    {action === "settings" || action === "instructions" ? copy.settingsHint(platform) : copy.why}
                  </p>
                  {(action === "request" || action === "settings") && (
                    <Button variant="secondary" loading={busy === permission} onClick={() => void run(permission)}>
                      {action === "request" ? "Разрешить" : "Открыть настройки"}
                    </Button>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </Modal>
  );
}
