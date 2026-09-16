import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Spinner, VerifiedName, toast } from "@shared/ui";
import { adminApi } from "@shared/api/endpoints";
import { formatRelative } from "@shared/lib/format";

// The verification mark: find an account or a group/community, give the mark
// or take it away. Every change lands in the audit log (the "Аудит" tab) with
// who did it and when; the server does the recording, not this screen.

function useDebounced(value: string, ms = 300) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

export function VerificationTab() {
  const [query, setQuery] = useState("");
  const q = useDebounced(query.trim());
  const qc = useQueryClient();
  const key = ["admin", "verification", q] as const;
  const { data, isPending } = useQuery({
    queryKey: key,
    queryFn: () => adminApi.searchVerification(q),
  });

  const refresh = () => qc.invalidateQueries({ queryKey: ["admin", "verification"] });

  const setUser = useMutation({
    mutationFn: (a: { id: string; verified: boolean }) => adminApi.setUserVerified(a.id, a.verified),
    onSuccess: (_r, a) => {
      toast.success(a.verified ? "Галочка выдана" : "Галочка снята");
      void refresh();
    },
    onError: () => toast.error("Не удалось изменить верификацию"),
  });
  const setGroup = useMutation({
    mutationFn: (a: { id: string; verified: boolean }) => adminApi.setGroupVerified(a.id, a.verified),
    onSuccess: (_r, a) => {
      toast.success(a.verified ? "Галочка выдана" : "Галочка снята");
      void refresh();
    },
    onError: () => toast.error("Не удалось изменить верификацию"),
  });

  return (
    <div className="admin-verify">
      <input
        className="ui-input"
        placeholder="Логин, имя или название сообщества"
        aria-label="Поиск для верификации"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />

      {isPending ? (
        <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
          <Spinner size={24} />
        </div>
      ) : (
        <>
          <h3 className="admin-verify__heading">Пользователи</h3>
          {data?.users.length === 0 && <p className="admin-verify__empty">Никого не найдено</p>}
          <ul className="admin-verify__list">
            {data?.users.map((u) => (
              <li key={u.id} className="admin-verify__row">
                <Avatar name={u.displayName} url={u.avatarUrl} size={34} />
                <div className="admin-verify__who">
                  <VerifiedName name={u.displayName} verified={!!u.verifiedAt} />
                  <span className="admin-verify__sub">
                    @{u.username}
                    {u.verifiedAt && ` · с ${formatRelative(u.verifiedAt)}`}
                  </span>
                </div>
                <Button
                  variant={u.verifiedAt ? "ghost" : "secondary"}
                  loading={setUser.isPending && setUser.variables?.id === u.id}
                  onClick={() => setUser.mutate({ id: u.id, verified: !u.verifiedAt })}
                >
                  {u.verifiedAt ? "Снять" : "Выдать"}
                </Button>
              </li>
            ))}
          </ul>

          <h3 className="admin-verify__heading">Группы и сообщества</h3>
          {data?.groups.length === 0 && <p className="admin-verify__empty">Ничего не найдено</p>}
          <ul className="admin-verify__list">
            {data?.groups.map((g) => (
              <li key={g.id} className="admin-verify__row">
                <Avatar name={g.name} url={g.avatarUrl} size={34} />
                <div className="admin-verify__who">
                  <VerifiedName name={g.name} verified={!!g.verifiedAt} subject="group" />
                  <span className="admin-verify__sub">
                    {g.kind === "community" ? "Сообщество" : "Группа"}
                    {g.verifiedAt && ` · с ${formatRelative(g.verifiedAt)}`}
                  </span>
                </div>
                <Button
                  variant={g.verifiedAt ? "ghost" : "secondary"}
                  loading={setGroup.isPending && setGroup.variables?.id === g.id}
                  onClick={() => setGroup.mutate({ id: g.id, verified: !g.verifiedAt })}
                >
                  {g.verifiedAt ? "Снять" : "Выдать"}
                </Button>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}
