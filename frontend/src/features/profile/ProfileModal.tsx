import { useEffect, useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { roleLabel } from "@shared/api/types";
import { authApi, usersApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { useThemeStore, type Theme } from "@shared/store/theme";
import { disablePush, enablePush, pushEnabled, pushSupported } from "@shared/lib/push";
import { useNotificationSettings, useUpdateNotificationSettings } from "@entities/notif-prefs/queries";
import type { GroupNotifyMode } from "@shared/api/endpoints";
import { AvatarCropper } from "./AvatarCropper";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function ProfileModal({ open, onClose }: Props) {
  const user = useAuthStore((s) => s.user);
  const setUser = useAuthStore((s) => s.setUser);
  const logout = useAuthStore((s) => s.logout);

  const [displayName, setDisplayName] = useState(user?.displayName ?? "");
  const [username, setUsername] = useState(user?.username ?? "");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [pushOn, setPushOn] = useState(false);
  const [pushBusy, setPushBusy] = useState(false);

  useEffect(() => {
    if (open && pushSupported()) void pushEnabled().then(setPushOn);
  }, [open]);

  const togglePush = async () => {
    setPushBusy(true);
    try {
      if (pushOn) {
        await disablePush();
        setPushOn(false);
        toast.success("Push-уведомления отключены");
      } else {
        const ok = await enablePush();
        setPushOn(ok);
        toast[ok ? "success" : "error"](ok ? "Push-уведомления включены" : "Не удалось включить push");
      }
    } catch {
      toast.error("Не удалось изменить push-уведомления");
    } finally {
      setPushBusy(false);
    }
  };

  if (!user) return null;

  const saveProfile = async () => {
    const fields: { displayName?: string; username?: string } = {};
    const trimmedName = displayName.trim();
    if (trimmedName !== user.displayName) {
      if (trimmedName.length < 1 || trimmedName.length > 64) {
        toast.error("Отображаемое имя: 1–64 символа");
        return;
      }
      fields.displayName = trimmedName;
    }
    if (username !== user.username) {
      if (!/^[A-Za-z0-9_]{3,32}$/.test(username)) {
        toast.error("Логин: 3–32 символа, буквы/цифры/подчёркивание");
        return;
      }
      fields.username = username;
    }
    if (!fields.displayName && !fields.username) {
      toast.error("Нет изменений");
      return;
    }
    setBusy(true);
    try {
      const { user: updated } = await usersApi.updateProfile(fields);
      setUser(updated);
      toast.success("Профиль обновлён");
    } catch {
      toast.error("Не удалось обновить профиль (возможно, логин занят)");
    } finally {
      setBusy(false);
    }
  };

  const uploadAvatar = async (blob: Blob) => {
    const { user: updated } = await usersApi.uploadAvatar(blob);
    setUser(updated);
  };

  const changePassword = async () => {
    if (newPassword.length < 12) {
      toast.error("Новый пароль — минимум 12 символов");
      return;
    }
    setBusy(true);
    try {
      await authApi.changePassword(currentPassword, newPassword);
      setCurrentPassword("");
      setNewPassword("");
      toast.success("Пароль изменён. Другие сессии завершены");
    } catch {
      toast.error("Не удалось изменить пароль");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal open={open} title="Профиль" onClose={onClose}>
      <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
        <AvatarCropper name={user.displayName} url={user.avatarUrl} size={56} onUpload={uploadAvatar} />
        <div>
          <div style={{ fontWeight: 640, fontSize: 17 }}>{user.displayName}</div>
          <div style={{ color: "var(--color-text-secondary)", fontSize: 14 }}>{roleLabel(user.roleLevel)}</div>
        </div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
        <Input label="Отображаемое имя" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
        <Input label="Логин" value={username} onChange={(e) => setUsername(e.target.value)} />
        <Button variant="secondary" onClick={saveProfile} loading={busy}>
          Сохранить профиль
        </Button>
      </div>

      <ThemeSwitcher />

      <div style={{ borderTop: "1px solid var(--color-border)", paddingTop: 16, display: "flex", flexDirection: "column", gap: 12 }}>
        <div style={{ fontWeight: 600, fontSize: 15 }}>Сменить пароль</div>
        <Input
          label="Текущий пароль"
          type="password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
        />
        <Input label="Новый пароль" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
        <Button variant="secondary" onClick={changePassword} loading={busy}>
          Изменить пароль
        </Button>
      </div>

      {pushSupported() && (
        <div style={{ borderTop: "1px solid var(--color-border)", paddingTop: 16, display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
          <div>
            <div style={{ fontWeight: 600, fontSize: 15 }}>Push-уведомления</div>
            <div style={{ color: "var(--color-text-secondary)", fontSize: 13 }}>
              О новых упоминаниях, когда вкладка закрыта
            </div>
          </div>
          <Button variant="secondary" onClick={() => void togglePush()} loading={pushBusy}>
            {pushOn ? "Отключить" : "Включить"}
          </Button>
        </div>
      )}

      <NotificationSettingsSection />

      <Button variant="danger" block onClick={() => void logout()}>
        Выйти из аккаунта
      </Button>
    </Modal>
  );
}

// Theme switcher (design handoff): a physical rotary knob ("manettino")
// centered between two columns of labels. The indicator rests at 3 o'clock and
// the knob rotates to point at the active label — the left column (Стекло /
// Luce / Аврора) is pointed at by rotating left, the right column (Cyber /
// Windows 95 / Matrix) by rotating right. Clicking the knob cycles through all
// six; clicking a label selects it directly.
type ThemeOption = { id: Theme; label: string; angle: number; col: "left" | "right" };
const THEME_OPTIONS: ThemeOption[] = [
  { id: "glass", label: "Стекло", angle: -150, col: "left" },
  { id: "luce", label: "Luce", angle: 180, col: "left" },
  { id: "aurora", label: "Аврора", angle: 150, col: "left" },
  { id: "cyber", label: "Cyber", angle: -30, col: "right" },
  { id: "xp", label: "Windows 95", angle: 0, col: "right" },
  { id: "matrix", label: "Matrix", angle: 30, col: "right" },
];

function GearIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" />
    </svg>
  );
}

function ThemeSwitcher() {
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const cycleTheme = useThemeStore((s) => s.cycleTheme);
  const angle = THEME_OPTIONS.find((o) => o.id === theme)?.angle ?? 0;

  const labelBtn = (o: ThemeOption) => (
    <button
      key={o.id}
      type="button"
      className={`theme-knob__label${theme === o.id ? " theme-knob__label--active" : ""}`}
      onClick={() => setTheme(o.id)}
    >
      {o.label}
    </button>
  );

  return (
    <div className="profile-section">
      <div className="profile-section__label">
        <GearIcon />
        Оформление
      </div>
      <div className="theme-knob">
        <div className="theme-knob__labels theme-knob__labels--left">
          {THEME_OPTIONS.filter((o) => o.col === "left").map(labelBtn)}
        </div>
        <button type="button" className="theme-knob__dial" onClick={cycleTheme} aria-label="Переключить тему">
          <span className="theme-knob__face" style={{ transform: `rotate(${angle}deg)` }}>
            <span className="theme-knob__dimple" />
            <span className="theme-knob__pointer" />
          </span>
        </button>
        <div className="theme-knob__labels theme-knob__labels--right">
          {THEME_OPTIONS.filter((o) => o.col === "right").map(labelBtn)}
        </div>
      </div>
    </div>
  );
}

const GROUP_MODE_LABELS: Record<GroupNotifyMode, string> = {
  all: "Все сообщения",
  mentions_only: "Только упоминания",
  none: "Отключены",
};

function NotificationSettingsSection() {
  const { data: settings } = useNotificationSettings();
  const update = useUpdateNotificationSettings();
  if (!settings) return null;

  const set = (patch: Partial<typeof settings>) =>
    update.mutate({ ...settings, ...patch }, { onError: () => toast.error("Не удалось сохранить настройки") });

  return (
    <div style={{ borderTop: "1px solid var(--color-border)", paddingTop: 16, display: "flex", flexDirection: "column", gap: 12 }}>
      <div style={{ fontWeight: 600, fontSize: 15 }}>Уведомления</div>

      <label style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
        <span style={{ fontSize: 14 }}>Звук</span>
        <input type="checkbox" checked={settings.sound} onChange={(e) => set({ sound: e.target.checked })} />
      </label>
      <label style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
        <span style={{ fontSize: 14 }}>Показывать превью текста</span>
        <input type="checkbox" checked={settings.preview} onChange={(e) => set({ preview: e.target.checked })} />
      </label>
      <label style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
        <span style={{ fontSize: 14 }}>Уведомления в группах</span>
        <select
          className="ui-input"
          style={{ width: "auto" }}
          value={settings.groupMode}
          onChange={(e) => set({ groupMode: e.target.value as GroupNotifyMode })}
        >
          {(Object.keys(GROUP_MODE_LABELS) as GroupNotifyMode[]).map((m) => (
            <option key={m} value={m}>
              {GROUP_MODE_LABELS[m]}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}
