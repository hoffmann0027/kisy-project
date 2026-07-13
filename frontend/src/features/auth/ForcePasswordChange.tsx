import { useState } from "react";
import { authApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { Button, Input, toast } from "@shared/ui";
import { ApiError } from "@shared/api/envelope";

// ForcePasswordChange is a blocking screen shown when the signed-in account
// still carries a seeded/administratively-reset password (mustChangePassword).
// It cannot be dismissed: the app is unreachable until a new password is set,
// closing the window where a shared bootstrap password grants Level-1 access.
export function ForcePasswordChange() {
  const user = useAuthStore((s) => s.user);
  const setUser = useAuthStore((s) => s.setUser);
  const logout = useAuthStore((s) => s.logout);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (newPassword.length < 12) {
      toast.error("Новый пароль — минимум 12 символов");
      return;
    }
    if (newPassword !== confirm) {
      toast.error("Пароли не совпадают");
      return;
    }
    if (newPassword === currentPassword) {
      toast.error("Новый пароль должен отличаться от текущего");
      return;
    }
    setBusy(true);
    try {
      await authApi.changePassword(currentPassword, newPassword);
      // Clear the flag locally so the gate opens without a round-trip.
      if (user) setUser({ ...user, mustChangePassword: false });
      toast.success("Пароль изменён");
    } catch (e) {
      toast.error(e instanceof ApiError && e.status === 401 ? "Неверный текущий пароль" : "Не удалось изменить пароль");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-screen">
      <div className="auth-card glass-surface">
        <div className="auth-brand">
          <h1 className="auth-title">Смена пароля</h1>
          <p className="auth-subtitle">
            Для этой учётной записи задан временный пароль. Задайте новый, чтобы продолжить.
          </p>
        </div>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void submit();
          }}
          style={{ display: "flex", flexDirection: "column", gap: 12 }}
        >
          <Input
            label="Текущий пароль"
            type="password"
            autoComplete="current-password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            autoFocus
          />
          <Input
            label="Новый пароль (мин. 12 символов)"
            type="password"
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
          />
          <Input
            label="Повторите новый пароль"
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <Button type="submit" block loading={busy}>
            Сохранить и войти
          </Button>
        </form>
        <Button variant="ghost" block onClick={() => void logout()}>
          Выйти
        </Button>
      </div>
    </div>
  );
}
