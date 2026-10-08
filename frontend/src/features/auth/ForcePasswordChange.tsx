import { useState } from "react";
import { authApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { Button, Input, toast } from "@shared/ui";
import { passwordProblem, passwordRuleText } from "@shared/lib/password";
import { midSentence } from "@shared/lib/format";
import { ApiError } from "@shared/api/envelope";
import { t } from "@shared/i18n";

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
    const problem = passwordProblem(newPassword);
    if (problem) {
      toast.error(t("account.password.newProblem", { problem: midSentence(problem) }));
      return;
    }
    if (newPassword !== confirm) {
      toast.error(t("account.password.mismatch"));
      return;
    }
    if (newPassword === currentPassword) {
      toast.error(t("account.password.mustDiffer"));
      return;
    }
    setBusy(true);
    try {
      await authApi.changePassword(currentPassword, newPassword);
      // Clear the flag locally so the gate opens without a round-trip.
      if (user) setUser({ ...user, mustChangePassword: false });
      toast.success(t("account.password.changed"));
    } catch (e) {
      toast.error(
        e instanceof ApiError && e.status === 401 ? t("account.password.wrongCurrent") : t("account.password.changeFailed"),
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-screen">
      <div className="auth-card glass-surface">
        <div className="auth-brand">
          <h1 className="auth-title">{t("account.passwordGate.title")}</h1>
          <p className="auth-subtitle">{t("account.passwordGate.lead")}</p>
        </div>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void submit();
          }}
          style={{ display: "flex", flexDirection: "column", gap: 12 }}
        >
          <Input
            label={t("account.password.current")}
            type="password"
            autoComplete="current-password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            autoFocus
          />
          <Input
            label={t("account.password.new")}
            hint={passwordRuleText()}
            type="password"
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
          />
          <Input
            label={t("account.passwordGate.confirm")}
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <Button type="submit" block loading={busy}>
            {t("account.passwordGate.submit")}
          </Button>
        </form>
        <Button variant="ghost" block onClick={() => void logout()}>
          {t("account.gate.signOut")}
        </Button>
      </div>
    </div>
  );
}
