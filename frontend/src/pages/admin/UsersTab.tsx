import { useEffect, useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Input, Modal, Spinner, toast } from "@shared/ui";
import { adminApi } from "@shared/api/endpoints";
import { ROLE_LABELS, roleLabel, type AdminUserFilter, type User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { PASSWORD_RULE_TEXT, passwordProblem } from "@shared/lib/password";

const PAGE = 100;

function useDebounced(value: string, ms = 300) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

// The CEO's user table. It used to load the newest hundred accounts and stop:
// anyone older could not be found. Search and filters now run on the server
// (GET /admin/users?q=&role=&status=), the way "Верификация" searches.
export function UsersTab() {
  const me = useAuthStore((s) => s.user!);
  const qc = useQueryClient();
  const [query, setQuery] = useState("");
  const [role, setRole] = useState("");
  const [status, setStatus] = useState<"" | "active" | "inactive">("");
  const q = useDebounced(query.trim());
  const filter: AdminUserFilter = { q: q || undefined, role: role || undefined, status: status || undefined };
  const { data: pages, isPending, hasNextPage, fetchNextPage, isFetchingNextPage } = useInfiniteQuery({
    queryKey: ["admin", "users", filter],
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => (await adminApi.users(filter, PAGE, pageParam)).users,
    getNextPageParam: (last, all) => (last.length === PAGE ? all.length * PAGE : undefined),
  });
  const data = pages?.pages.flat();
  const filtered = !!(q || role || status);

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

  const filters = (
    <div className="admin-users__filters">
      <input
        className="ui-input"
        placeholder="Логин или имя"
        aria-label="Поиск пользователей"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <select className="ui-input" aria-label="Роль" value={role} onChange={(e) => setRole(e.target.value)}>
        <option value="">Все роли</option>
        <option value="basic">Без уровня (basic)</option>
        {Object.entries(ROLE_LABELS).map(([lvl, label]) => (
          <option key={lvl} value={lvl}>
            {lvl}. {label}
          </option>
        ))}
      </select>
      <select
        className="ui-input"
        aria-label="Статус"
        value={status}
        onChange={(e) => setStatus(e.target.value as "" | "active" | "inactive")}
      >
        <option value="">Любой статус</option>
        <option value="active">Активные</option>
        <option value="inactive">Отключённые</option>
      </select>
    </div>
  );

  if (isPending) {
    return (
      <>
        {filters}
        <div style={{ display: "flex", justifyContent: "center", padding: 40 }}>
          <Spinner size={28} />
        </div>
      </>
    );
  }

  return (
    <>
      {filters}
      <p className="admin-users__count">
        {data?.length ?? 0}
        {hasNextPage ? "+" : ""} {filtered ? "найдено" : "всего"}
      </p>
      {data?.length === 0 && <p className="admin-verify__empty">Никого не найдено</p>}
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
                ) : u.accountKind === "basic" ? (
                  // Not a disabled dropdown of levels this account could have:
                  // it could have none. Granting one would make it an invited
                  // account that nobody invited, and the server refuses it.
                  <span
                    style={{ color: "var(--color-text-tertiary)" }}
                    title="Аккаунт зарегистрирован без приглашения и не входит в иерархию уровней"
                  >
                    Без уровня
                  </span>
                ) : (
                  <select
                    className="role-select"
                    value={u.roleLevel ?? ""}
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
      {hasNextPage && (
        <Button variant="ghost" loading={isFetchingNextPage} onClick={() => void fetchNextPage()}>
          Показать ещё
        </Button>
      )}

      <ResetPasswordModal user={resetFor} onClose={() => setResetFor(null)} />
    </>
  );
}

function ResetPasswordModal({ user, onClose }: { user: User | null; onClose: () => void }) {
  const [pw, setPw] = useState("");
  // The CEO's reset obeys the same rule as sign-up: the server refuses a
  // weaker password, and the form used to promise otherwise (audit D-14).
  const problem = passwordProblem(pw);
  const reset = useMutation({
    mutationFn: (id: string) => adminApi.resetPassword(id, pw),
    onSuccess: () => {
      toast.success("Пароль сброшен, сессии пользователя завершены");
      setPw("");
      onClose();
    },
    onError: () => toast.error(`Не удалось сбросить пароль (${PASSWORD_RULE_TEXT})`),
  });

  return (
    <Modal open={!!user} title={`Сброс пароля: ${user?.displayName ?? ""}`} onClose={onClose}>
      <Input
        label="Новый пароль"
        type="text"
        value={pw}
        onChange={(e) => setPw(e.target.value)}
        hint={PASSWORD_RULE_TEXT}
        error={pw ? (problem ?? undefined) : undefined}
      />
      <Button block disabled={!!problem} loading={reset.isPending} onClick={() => user && reset.mutate(user.id)}>
        Сбросить пароль
      </Button>
    </Modal>
  );
}
