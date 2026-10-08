import type { ReactNode } from "react";
import { Icon } from "@shared/ui/icons";
import { t } from "@shared/i18n";
import type { AppPermission, PermissionState, Platform } from "./sequence";

// What each permission is for, in the user's words. Shared by the onboarding
// and the "Разрешения" screen in the profile, so the two never explain the same
// thing differently.

export interface PermissionCopy {
  icon: ReactNode;
  title: string;
  /** Why the app wants it — shown before the system dialog. */
  why: string;
  /** Shown after one refusal, when the dialog can still be shown again. */
  again: string;
  /** Where to go when only Settings can change it. */
  settingsHint: (platform: Platform) => string;
}

/** The copy for one permission, in the language on screen (built on call, never at import). */
export function permissionCopy(permission: AppPermission): PermissionCopy {
  switch (permission) {
    case "notifications":
      return {
        icon: <Icon.Bell size={40} />,
        title: t("account.permissions.notificationsTitle"),
        why: t("account.permissions.notificationsWhy"),
        again: t("account.permissions.notificationsAgain"),
        settingsHint: (platform) =>
          platform === "native"
            ? t("account.permissions.notificationsSettingsNative")
            : t("account.permissions.notificationsSettingsWeb"),
      };
    case "microphone":
      return {
        icon: <Icon.Mic size={40} />,
        title: t("account.permissions.microphoneTitle"),
        why: t("account.permissions.microphoneWhy"),
        again: t("account.permissions.microphoneAgain"),
        settingsHint: () => t("account.permissions.microphoneSettings"),
      };
    case "fullScreenIntent":
      return {
        icon: <Icon.Phone size={40} />,
        title: t("account.permissions.fullScreenTitle"),
        why: t("account.permissions.fullScreenWhy"),
        again: t("account.permissions.fullScreenAgain"),
        settingsHint: () => t("account.permissions.fullScreenSettings"),
      };
  }
}

export function stateLabel(state: PermissionState | undefined): string {
  switch (state) {
    case "granted":
      return t("account.permissions.stateGranted");
    case "denied":
      return t("account.permissions.stateDenied");
    case "prompt":
    case "prompt-with-rationale":
      return t("account.permissions.statePrompt");
    case "unsupported":
      return t("account.permissions.stateUnsupported");
    default:
      return "…";
  }
}
