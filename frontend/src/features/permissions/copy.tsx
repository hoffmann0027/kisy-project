import type { ReactNode } from "react";
import { Icon } from "@shared/ui/icons";
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

export const PERMISSION_COPY: Record<AppPermission, PermissionCopy> = {
  notifications: {
    icon: <Icon.Bell size={40} />,
    title: "Уведомления",
    why: "Чтобы вы узнавали о новых сообщениях, упоминаниях и звонках, даже когда KISY закрыт.",
    again: "Без уведомлений о входящем звонке можно узнать, только если приложение открыто.",
    settingsHint: (platform) =>
      platform === "native"
        ? "Android больше не покажет этот запрос. Включите уведомления для KISY в настройках — это займёт пару секунд."
        : "Браузер больше не покажет этот запрос. Разрешите уведомления для этого сайта в настройках браузера — обычно это значок замка слева от адреса.",
  },
  microphone: {
    icon: <Icon.Mic size={40} />,
    title: "Микрофон",
    why: "Нужен для звонков и голосовых сообщений. Он включается, только когда вы сами звоните, отвечаете или записываете голосовое.",
    again: "Без микрофона вас не услышат в звонке, а голосовое не запишется.",
    settingsHint: () =>
      "Android больше не покажет этот запрос. Откройте настройки, раздел «Разрешения», и разрешите KISY микрофон.",
  },
  fullScreenIntent: {
    icon: <Icon.Phone size={40} />,
    title: "Экран входящего звонка",
    why: "Чтобы звонок открывался на весь экран, даже когда телефон заблокирован, — а не прятался в шторке.",
    again: "Без этого звонок на заблокированном телефоне придёт маленьким уведомлением.",
    settingsHint: () =>
      "Android разрешает это только в настройках. Откроется страница KISY — включите переключатель и вернитесь в приложение.",
  },
};

export function stateLabel(state: PermissionState | undefined): string {
  switch (state) {
    case "granted":
      return "Разрешено";
    case "denied":
      return "Запрещено";
    case "prompt":
    case "prompt-with-rationale":
      return "Не спрашивали";
    case "unsupported":
      return "Недоступно";
    default:
      return "…";
  }
}
