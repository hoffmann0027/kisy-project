import { useState } from "react";
import { usersApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { Button, Input, toast } from "@shared/ui";
import {
  displayNameErrorMessage,
  displayNameProblem,
  normalizeDisplayName,
} from "@shared/lib/displayName";
import { t } from "@shared/i18n";

// ForceDisplayNameChange blocks the app for an account whose name stopped
// being allowed when display names became unique and letters-only
// (migration 46). Nobody's name was rewritten behind their back; instead the
// owner picks a new one here, once, before anything else loads.
//
// Why this account is here is told honestly: either the old name breaks the
// rule, or it is valid and someone who registered earlier already has it.
export function ForceDisplayNameChange() {
  const user = useAuthStore((s) => s.user);
  const setUser = useAuthStore((s) => s.setUser);
  const logout = useAuthStore((s) => s.logout);
  const [name, setName] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  if (!user) return null;
  const oldNameBreaksRule = displayNameProblem(user.displayName) !== null;

  const submit = async () => {
    const problem = displayNameProblem(name);
    if (problem) {
      setError(problem);
      return;
    }
    setBusy(true);
    try {
      const { user: updated } = await usersApi.updateProfile({ displayName: normalizeDisplayName(name) });
      // The server clears the flag in the same update; the gate opens on it.
      setUser(updated);
      toast.success(t("account.renameGate.saved"));
    } catch (e) {
      setError(displayNameErrorMessage(e) ?? t("account.renameGate.saveFailed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-screen">
      <div className="auth-card glass-surface">
        <div className="auth-brand">
          <h1 className="auth-title">{t("account.renameGate.title")}</h1>
          <p className="auth-subtitle">
            {t(oldNameBreaksRule ? "account.renameGate.invalid" : "account.renameGate.taken", { name: user.displayName })}{" "}
            {t("account.renameGate.pickNew", { username: user.username })}
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
            label={t("account.renameGate.newName")}
            placeholder={t("account.fields.namePlaceholder")}
            autoComplete="name"
            autoFocus
            value={name}
            error={error}
            onChange={(e) => {
              setName(e.target.value);
              setError(undefined);
            }}
          />
          <Button type="submit" block loading={busy}>
            {t("account.renameGate.submit")}
          </Button>
        </form>
        <Button variant="ghost" block onClick={() => void logout()}>
          {t("account.gate.signOut")}
        </Button>
      </div>
    </div>
  );
}
