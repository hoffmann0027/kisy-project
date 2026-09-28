import { useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { usersApi } from "@shared/api/endpoints";
import { userFacingError } from "@shared/api/envelope";

// Deleting your own account. Two deliberate steps — the password and the word
// — because this cannot be undone: there is no grace period and no restore.
// What goes and what stays is spelled out here rather than in a help page
// nobody opens.

export const CONFIRM_WORD = "УДАЛИТЬ";

export function DeleteAccountDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);

  const ready = password.length > 0 && confirm.trim() === CONFIRM_WORD;

  const submit = async () => {
    if (!ready || busy) return;
    setBusy(true);
    try {
      await usersApi.deleteAccount(password, confirm.trim());
      // The server has ended the session; reloading drops every cache this
      // page holds for the account that no longer exists.
      window.location.replace("/login?deleted=1");
    } catch (e) {
      toast.error(userFacingError(e, "Не удалось удалить аккаунт"));
      setBusy(false);
    }
  };

  return (
    <Modal open={open} title="Удалить аккаунт" onClose={onClose}>
      <p className="delete-account__lead">Это действие необратимо — отменить удаление нельзя.</p>

      <p className="delete-account__group">Будет удалено:</p>
      <ul className="delete-account__list">
        <li>пароль, все сеансы и устройства;</li>
        <li>ключи шифрования и уведомления;</li>
        <li>ваши файлы, заметки и настройки;</li>
        <li>тексты ваших личных сообщений — и у собеседника тоже.</li>
      </ul>

      <p className="delete-account__group">Останется:</p>
      <ul className="delete-account__list">
        <li>ваши сообщения в группах и записи в сообществах — от «Удалённого аккаунта»;</li>
        <li>группы и сообщества, которыми вы управляли, перейдут следующему по управлению;</li>
        <li>ваш логин не достанется никому другому.</li>
      </ul>

      <Input
        label="Пароль"
        type="password"
        autoComplete="current-password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />
      <Input
        label={`Введите ${CONFIRM_WORD}, чтобы подтвердить`}
        placeholder={CONFIRM_WORD}
        value={confirm}
        onChange={(e) => setConfirm(e.target.value)}
      />

      <div className="delete-account__actions">
        <Button variant="secondary" block onClick={onClose} disabled={busy}>
          Отмена
        </Button>
        <Button variant="danger" block onClick={() => void submit()} disabled={!ready} loading={busy}>
          Удалить навсегда
        </Button>
      </div>
    </Modal>
  );
}
