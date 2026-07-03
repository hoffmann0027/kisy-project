import { useState } from "react";
import { Avatar, Button, Input, Modal, toast } from "@shared/ui";
import { roleLabel } from "@shared/api/types";
import { authApi, usersApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function ProfileModal({ open, onClose }: Props) {
  const user = useAuthStore((s) => s.user);
  const setUser = useAuthStore((s) => s.setUser);
  const logout = useAuthStore((s) => s.logout);

  const [username, setUsername] = useState(user?.username ?? "");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [busy, setBusy] = useState(false);

  if (!user) return null;

  const saveUsername = async () => {
    if (!/^[A-Za-z0-9_]{3,32}$/.test(username)) {
      toast.error("Имя: 3–32 символа, буквы/цифры/подчёркивание");
      return;
    }
    setBusy(true);
    try {
      const { user: updated } = await usersApi.updateUsername(username);
      setUser(updated);
      toast.success("Имя обновлено");
    } catch {
      toast.error("Не удалось обновить имя (возможно, занято)");
    } finally {
      setBusy(false);
    }
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
        <Avatar name={user.displayName} url={user.avatarUrl} size={56} />
        <div>
          <div style={{ fontWeight: 640, fontSize: 17 }}>{user.displayName}</div>
          <div style={{ color: "var(--color-text-secondary)", fontSize: 14 }}>{roleLabel(user.roleLevel)}</div>
        </div>
      </div>

      <div style={{ display: "flex", gap: 8, alignItems: "flex-end" }}>
        <div style={{ flex: 1 }}>
          <Input label="Имя пользователя" value={username} onChange={(e) => setUsername(e.target.value)} />
        </div>
        <Button variant="secondary" onClick={saveUsername} loading={busy}>
          Сохранить
        </Button>
      </div>

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

      <Button variant="danger" block onClick={() => void logout()}>
        Выйти из аккаунта
      </Button>
    </Modal>
  );
}
