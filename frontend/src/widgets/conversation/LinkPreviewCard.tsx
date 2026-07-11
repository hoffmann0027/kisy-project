// Link preview card under a message (stage E). Lazily fetches OpenGraph
// metadata for the message's first URL through the SSRF-guarded server
// endpoint. In E2EE chats the server never sees the URL, so the lookup only
// runs on an explicit user action (allowPreview) — the card offers a button
// instead of auto-fetching.
import { useState } from "react";
import { linkPreviewApi } from "@shared/api/endpoints";
import { useLinkPreview } from "@entities/link-preview/queries";

interface Props {
  url: string;
  /** True for plaintext chats (auto-fetch is safe: server already sees the
   * text). False for E2EE chats until the user opts in. */
  autoFetch: boolean;
}

export function LinkPreviewCard({ url, autoFetch }: Props) {
  const [optedIn, setOptedIn] = useState(false);
  const enabled = autoFetch || optedIn;
  const { data, isPending, isError } = useLinkPreview(url, enabled);

  if (!enabled) {
    return (
      <button className="linkcard linkcard--optin" onClick={() => setOptedIn(true)}>
        Показать превью ссылки
        <span className="linkcard__hint">Запрос уйдёт на сервер (в зашифрованном чате)</span>
      </button>
    );
  }
  if (isError) return null;
  if (isPending || !data) return <div className="linkcard linkcard--skeleton" />;
  if (!data.title && !data.description) return null;

  return (
    <a className="linkcard" href={data.url} target="_blank" rel="noopener noreferrer nofollow">
      {data.imageUrl && (
        <img className="linkcard__img" src={linkPreviewApi.imageProxyUrl(data.imageUrl)} alt="" loading="lazy" />
      )}
      <div className="linkcard__body">
        {data.siteName && <div className="linkcard__site">{data.siteName}</div>}
        {data.title && <div className="linkcard__title">{data.title}</div>}
        {data.description && <div className="linkcard__desc">{data.description}</div>}
      </div>
    </a>
  );
}
