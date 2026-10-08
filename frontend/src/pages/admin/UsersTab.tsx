import { useEffect, useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Input, Modal, Spinner, toast } from "@shared/ui";
import { adminApi } from "@shared/api/endpoints";
import { ROLE_LABELS, roleLabel, type AdminUserFilter, type User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { passwordProblem, passwordRuleText } from "@shared/lib/password";
import { t } from "@shared/i18n";

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
      toast.success(t("admin.users.roleChanged"));
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
    onError: () => toast.error(t("admin.users.roleChangeFailed")),
  });

  const toggleActive = useMutation({
    mutationFn: (u: User) => (u.isActive ? adminApi.deactivate(u.id) : adminApi.activate(u.id)),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "users"] }),
    onError: () => toast.error(t("admin.users.statusChangeFailed")),
  });

  const filters = (
    <div className="admin-users__filters">
      <input
        className="ui-input"
        placeholder={t("admin.users.searchPlaceholder")}
        aria-label={t("admin.users.searchLabel")}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <select className="ui-input" aria-label={t("admin.users.roleFilter")} value={role} onChange={(e) => setRole(e.target.value)}>
        <option value="">{t("admin.users.allRoles")}</option>
        <option value="basic">{t("admin.users.basicOption")}</option>
        {Object.entries(ROLE_LABELS).map(([lvl, label]) => (
          <option key={lvl} value={lvl}>
            {lvl}. {label}
          </option>
        ))}
      </select>
      <select
        className="ui-input"
        aria-label={t("admin.users.statusFilter")}
        value={status}
        onChange={(e) => setStatus(e.target.value as "" | "active" | "inactive")}
      >
        <option value="">{t("admin.users.anyStatus")}</option>
        <option value="active">{t("admin.users.statusActive")}</option>
        <option value="inactive">{t("admin.users.statusInactive")}</option>
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
        {t(filtered ? "admin.users.countFound" : "admin.users.countTotal", { n: `${data?.length ?? 0}${hasNextPage ? "+" : ""}` })}
      </p>
      {data?.length === 0 && <p className="admin-verify__empty">{t("admin.users.empty")}</p>}
      <table className="table">
        <thead>
          <tr>
            <th>{t("admin.users.colUser")}</th>
            <th>{t("admin.users.colRole")}</th>
            <th>{t("admin.users.colStatus")}</th>
            <th style={{ textAlign: "right" }}>{t("admin.users.colActions")}</th>
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
                    title={t("admin.users.noLevelHint")}
                  >
                    {t("admin.users.noLevel")}
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
                  {u.isActive ? t("admin.users.active") : t("admin.users.inactive")}
                </span>
              </td>
              <td style={{ textAlign: "right" }}>
                {u.id !== me.id && (
                  <div style={{ display: "inline-flex", gap: 8 }}>
                    <Button variant="ghost" onClick={() => setResetFor(u)}>
                      {t("admin.users.resetPassword")}
                    </Button>
                    <Button
                      variant={u.isActive ? "danger" : "secondary"}
                      onClick={() => toggleActive.mutate(u)}
                    >
                      {u.isActive ? t("admin.users.deactivate") : t("admin.users.activate")}
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
          {t("admin.users.loadMore")}
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
      toast.success(t("admin.users.resetDone"));
      setPw("");
      onClose();
    },
    onError: () => toast.error(t("admin.users.resetFailed", { rule: passwordRuleText() })),
  });

  return (
    <Modal open={!!user} title={t("admin.users.resetTitle", { name: user?.displayName ?? "" })} onClose={onClose}>
      <Input
        label={t("admin.users.newPassword")}
        type="text"
        value={pw}
        onChange={(e) => setPw(e.target.value)}
        hint={passwordRuleText()}
        error={pw ? (problem ?? undefined) : undefined}
      />
      <Button block disabled={!!problem} loading={reset.isPending} onClick={() => user && reset.mutate(user.id)}>
        {t("admin.users.resetPassword")}
      </Button>
    </Modal>
  );
}
