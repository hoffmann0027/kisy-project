import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Input, Modal, Spinner, toast } from "@shared/ui";
import { adminApi } from "@shared/api/endpoints";
import { ROLE_LABELS, roleLabel, type User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";

export function UsersTab() {
  const me = useAuthStore((s) => s.user!);
  const qc = useQueryClient();
  const { data, isPending } = useQuery({
    queryKey: ["admin", "users"],
    queryFn: async () => (await adminApi.users()).users,
  });

  const [resetFor, setResetFor] = useState<User | null>(null);

  const changeRole = useMutation({
    mutationFn: (args: { id: string; role: number }) => adminApi.changeRole(args.id, args.role),
    onSuccess: () => {
      toast.success("Роль изменена");
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
    onError: () => toast.error("Не удалось изменить роль"),
  });

  const toggleActive = useMutation({
    mutationFn: (u: User) => (u.isActive ? adminApi.deactivate(u.id) : adminApi.activate(u.id)),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "users"] }),
    onError: () => toast.error("Не удалось изменить статус"),
  });

  if (isPending) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 40 }}>
        <Spinner size={28} />
      </div>
    );
  }

  return (
    <>
      <table className="table">
        <thead>
          <tr>
            <th>Пользователь</th>
            <th>Роль</th>
            <th>Статус</th>
            <th style={{ textAlign: "right" }}>Действия</th>
          </tr>
        </thead>
        <tbody>
          {data?.map((u) => (
            <tr key={u.id}>
              <td>
                <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  <Avatar name={u.displayName} url={u.avatarUrl} size={34} />
                  <div>
                    <div style={{ fontWeight: 560 }}>{u.displayName}</div>
                    <div style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>@{u.username}</div>
                  </div>
                </div>
              </td>
              <td>
                {u.id === me.id ? (
                  <span>{roleLabel(u.roleLevel)}</span>
                ) : (
                  <select
                    className="role-select"
                    value={u.roleLevel}
                    onChange={(e) => changeRole.mutate({ id: u.id, role: Number(e.target.value) })}
                  >
                    {Object.entries(ROLE_LABELS).map(([lvl, label]) => (
                      <option key={lvl} value={lvl}>
                        {lvl}. {label}
                      </option>
                    ))}
                  </select>
                )}
              </td>
              <td>
                <span className={u.isActive ? "pill pill--active" : "pill pill--inactive"}>
                  {u.isActive ? "активен" : "отключён"}
                </span>
              </td>
              <td style={{ textAlign: "right" }}>
                {u.id !== me.id && (
                  <div style={{ display: "inline-flex", gap: 8 }}>
                    <Button variant="ghost" onClick={() => setResetFor(u)}>
                      Сбросить пароль
                    </Button>
                    <Button
                      variant={u.isActive ? "danger" : "secondary"}
                      onClick={() => toggleActive.mutate(u)}
                    >
                      {u.isActive ? "Отключить" : "Включить"}
                    </Button>
                  </div>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <ResetPasswordModal user={resetFor} onClose={() => setResetFor(null)} />
    </>
  );
}

function ResetPasswordModal({ user, onClose }: { user: User | null; onClose: () => void }) {
  const [pw, setPw] = useState("");
  const reset = useMutation({
    mutationFn: (id: string) => adminApi.resetPassword(id, pw),
    onSuccess: () => {
      toast.success("Пароль сброшен, сессии пользователя завершены");
      setPw("");
      onClose();
    },
    onError: () => toast.error("Не удалось сбросить пароль (минимум 12 символов)"),
  });

  return (
    <Modal open={!!user} title={`Сброс пароля: ${user?.displayName ?? ""}`} onClose={onClose}>
      <Input
        label="Новый пароль"
        type="text"
        value={pw}
        onChange={(e) => setPw(e.target.value)}
        placeholder="Минимум 12 символов"
      />
      <Button block loading={reset.isPending} onClick={() => user && reset.mutate(user.id)}>
        Сбросить пароль
      </Button>
    </Modal>
  );
}
