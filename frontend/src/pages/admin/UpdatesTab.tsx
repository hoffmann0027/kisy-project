import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Input, Spinner, toast } from "@shared/ui";
import { adminApi } from "@shared/api/endpoints";
import { userFacingError } from "@shared/api/envelope";
import type { AppBuildCount } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";

// "New Update": tell everyone a new version is out — what it is called, what
// changed, where to get it — and see how many people still run an older
// build. The server sends it to every active account as a notification with a
// push (POST /admin/releases).

const NOTES_MAX = 4000;

/** Accounts on the newest build seen, and on anything older. */
export function buildShare(versions: AppBuildCount[]): { latest: AppBuildCount | null; onLatest: number; older: number } {
  if (versions.length === 0) return { latest: null, onLatest: 0, older: 0 };
  const latest = versions.reduce((a, b) => (b.build > a.build ? b : a));
  const total = versions.reduce((s, v) => s + v.users, 0);
  return { latest, onLatest: latest.users, older: total - latest.users };
}

export function UpdatesTab() {
  const qc = useQueryClient();
  const { data, isPending } = useQuery({ queryKey: ["admin", "releases"], queryFn: () => adminApi.releases() });
  const [version, setVersion] = useState("");
  const [notes, setNotes] = useState("");
  const [link, setLink] = useState("");

  const linkOk = link.trim() === "" || /^https:\/\/[^\s/]+/.test(link.trim());
  const ready = version.trim() !== "" && notes.trim() !== "" && linkOk;

  const send = useMutation({
    mutationFn: () =>
      adminApi.sendRelease({ version: version.trim(), notes: notes.trim(), downloadUrl: link.trim() || undefined }),
    onSuccess: ({ release }) => {
      toast.success(`Версия ${release.version} объявлена: получателей — ${release.recipientCount}`);
      setVersion("");
      setNotes("");
      setLink("");
      void qc.invalidateQueries({ queryKey: ["admin", "releases"] });
    },
    onError: (e) => toast.error(userFacingError(e, "Не удалось объявить версию")),
  });

  const submit = () => {
    if (!ready || send.isPending) return;
    if (!window.confirm(`Отправить всем пользователям уведомление о версии ${version.trim()}?`)) return;
    send.mutate();
  };

  const share = buildShare(data?.versions ?? []);

  return (
    <div className="dash-updates">
      <section className="dash-card" aria-label="Объявить новую версию">
        <header className="dash-card__head">
          <h3 className="dash-card__title">Объявить новую версию</h3>
        </header>
        <Input label="Версия" placeholder="1.4.0" value={version} maxLength={32} onChange={(e) => setVersion(e.target.value)} />
        <div className="ui-field">
          <label className="ui-field__label" htmlFor="release-notes">
            Что нового
          </label>
          <textarea
            id="release-notes"
            className="ui-input announce-form__body"
            rows={5}
            maxLength={NOTES_MAX}
            placeholder={"— Звонки переживают смену сети\n— Новый логотип"}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </div>
        <Input
          label="Ссылка на скачивание (необязательно)"
          placeholder="https://…"
          value={link}
          onChange={(e) => setLink(e.target.value)}
          error={linkOk ? undefined : "Ссылка должна начинаться с https://"}
        />
        <p className="announce-form__hint">Уведомление с пушем получат все активные пользователи.</p>
        <Button disabled={!ready} loading={send.isPending} onClick={submit}>
          Отправить всем
        </Button>
      </section>

      <section className="dash-card" aria-label="Версии у пользователей">
        <header className="dash-card__head">
          <h3 className="dash-card__title">Версии приложения у пользователей</h3>
        </header>
        {isPending ? (
          <div className="dash-loading">
            <Spinner size={22} />
          </div>
        ) : share.latest ? (
          <>
            <p className="dash-updates__share">
              На новейшей сборке ({share.latest.version}): <strong>{share.onLatest}</strong> · на старых:{" "}
              <strong>{share.older}</strong>
            </p>
            <ul className="dash-list">
              {data!.versions.map((v) => (
                <li key={v.build} className="dash-list__row">
                  <div>
                    <div className="dash-list__title">{v.version}</div>
                    <div className="dash-list__sub">сборка {v.build}</div>
                  </div>
                  <span className="dash-list__time">{v.users} чел.</span>
                </li>
              ))}
            </ul>
          </>
        ) : (
          <p className="admin-verify__empty">
            Пока нет данных: версию сообщают сборки приложения начиная с этой, за последние 30 дней.
          </p>
        )}
      </section>

      <section className="dash-card" aria-label="Объявленные версии">
        <header className="dash-card__head">
          <h3 className="dash-card__title">Объявленные версии</h3>
        </header>
        {!isPending && (data?.releases.length ?? 0) === 0 && <p className="admin-verify__empty">Ещё ничего не объявляли</p>}
        <ul className="dash-list">
          {data?.releases.map((r) => (
            <li key={r.id} className="dash-release">
              <div className="dash-list__title">
                {r.version} <span className="dash-list__sub">· {formatRelative(r.createdAt)} · получателей: {r.recipientCount}</span>
              </div>
              <div className="dash-release__notes">{r.notes}</div>
              {r.downloadUrl && (
                <a className="dash-link" href={r.downloadUrl} target="_blank" rel="noopener noreferrer">
                  {r.downloadUrl}
                </a>
              )}
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
