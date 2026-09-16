import { useState } from "react";
import { usersApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { Button, Input, toast } from "@shared/ui";
import {
  displayNameErrorMessage,
  displayNameProblem,
  normalizeDisplayName,
} from "@shared/lib/displayName";

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
      toast.success("Имя сохранено");
    } catch (e) {
      setError(displayNameErrorMessage(e) ?? "Не удалось сохранить имя");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-screen">
      <div className="auth-card glass-surface">
        <div className="auth-brand">
          <h1 className="auth-title">Выберите имя</h1>
          <p className="auth-subtitle">
            {oldNameBreaksRule
              ? `Имя «${user.displayName}» больше не подходит: в имени теперь можно использовать только буквы и одиночные пробелы, от 2 до 40 символов.`
              : `Имя «${user.displayName}» уже есть у другого пользователя: теперь имена в KISY уникальны.`}{" "}
            Выберите новое, чтобы продолжить. Ваш логин @{user.username} не меняется.
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
            label="Новое имя"
            placeholder="Анна Смирнова"
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
            Сохранить и продолжить
          </Button>
        </form>
        <Button variant="ghost" block onClick={() => void logout()}>
          Выйти
        </Button>
      </div>
    </div>
  );
}
