import { useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { usersApi } from "@shared/api/endpoints";
import { userFacingError } from "@shared/api/envelope";
import { t } from "@shared/i18n";

// Deleting your own account. Two deliberate steps — the password and the word
// — because this cannot be undone: there is no grace period and no restore.
// What goes and what stays is spelled out here rather than in a help page
// nobody opens.
//
// The word is the one of the language on screen; the server accepts the word
// of every language it supports, so the word typed is sent, not a fixed one.

export function DeleteAccountDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const word = t("account.delete.confirmWord");

  const ready = password.length > 0 && confirm.trim() === word;

  const submit = async () => {
    if (!ready || busy) return;
    setBusy(true);
    try {
      await usersApi.deleteAccount(password, confirm.trim());
      // The server has ended the session; reloading drops every cache this
      // page holds for the account that no longer exists.
      window.location.replace("/login?deleted=1");
    } catch (e) {
      toast.error(userFacingError(e, t("account.delete.failed")));
      setBusy(false);
    }
  };

  return (
    <Modal open={open} title={t("account.delete.title")} onClose={onClose}>
      <p className="delete-account__lead">{t("account.delete.lead")}</p>

      <p className="delete-account__group">{t("account.delete.goes")}</p>
      <ul className="delete-account__list">
        <li>{t("account.delete.goesCredentials")}</li>
        <li>{t("account.delete.goesKeys")}</li>
        <li>{t("account.delete.goesFiles")}</li>
        <li>{t("account.delete.goesDirect")}</li>
      </ul>

      <p className="delete-account__group">{t("account.delete.stays")}</p>
      <ul className="delete-account__list">
        <li>{t("account.delete.staysPosts")}</li>
        <li>{t("account.delete.staysGroups")}</li>
        <li>{t("account.delete.staysUsername")}</li>
      </ul>

      <Input
        label={t("account.fields.password")}
        type="password"
        autoComplete="current-password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />
      <Input
        label={t("account.delete.typeToConfirm", { word })}
        placeholder={word}
        value={confirm}
        onChange={(e) => setConfirm(e.target.value)}
      />

      <div className="delete-account__actions">
        <Button variant="secondary" block onClick={onClose} disabled={busy}>
          {t("account.delete.cancel")}
        </Button>
        <Button variant="danger" block onClick={() => void submit()} disabled={!ready} loading={busy}>
          {t("account.delete.submit")}
        </Button>
      </div>
    </Modal>
  );
}
