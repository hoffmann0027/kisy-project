// Markdown-subset renderer (stage E). Tokenizes text into React nodes — never
// dangerouslySetInnerHTML — so message bodies can carry **bold**, _italic_,
// ~~strike~~, `code`, ```code blocks```, autolinks and @mentions without any
// HTML-injection surface. The source text is stored as-is; formatting is a
// render concern.
import { Fragment, type ReactNode } from "react";

const MENTION = /@[A-Za-z0-9_]{3,32}/;
const URL = /https?:\/\/[^\s<]+/;

// Inline formatting markers, tried by earliest position (recursive descent so
// nesting like **bold _italic_** renders correctly).
interface Marker {
  re: RegExp;
  wrap: (inner: ReactNode, key: number) => ReactNode;
}
const MARKERS: Marker[] = [
  { re: /`([^`]+)`/, wrap: (inner, k) => <code key={k} className="rt-code">{inner}</code> },
  { re: /\*\*([^*]+)\*\*/, wrap: (inner, k) => <strong key={k}>{inner}</strong> },
  { re: /~~([^~]+)~~/, wrap: (inner, k) => <del key={k}>{inner}</del> },
  { re: /_([^_]+)_/, wrap: (inner, k) => <em key={k}>{inner}</em> },
];

let keySeq = 0;

// renderLeaf splits plain text (no formatting markers) into autolinks and
// @mentions.
function renderLeaf(text: string): ReactNode[] {
  const out: ReactNode[] = [];
  let rest = text;
  const combined = new RegExp(`(${URL.source})|(${MENTION.source})`, "g");
  let last = 0;
  let m: RegExpExecArray | null;
  combined.lastIndex = 0;
  while ((m = combined.exec(rest)) !== null) {
    if (m.index > last) out.push(rest.slice(last, m.index));
    if (m[1]) {
      const href = m[1];
      out.push(
        <a key={keySeq++} href={href} target="_blank" rel="noopener noreferrer nofollow" className="rt-link">
          {href}
        </a>,
      );
    } else {
      out.push(<span key={keySeq++} className="mention">{m[2]}</span>);
    }
    last = m.index + m[0].length;
  }
  if (last < rest.length) out.push(rest.slice(last));
  return out;
}

// renderInline applies the earliest formatting marker, recursing into its
// content and the surrounding text.
function renderInline(text: string): ReactNode[] {
  let best: { marker: Marker; match: RegExpMatchArray } | null = null;
  for (const marker of MARKERS) {
    const match = text.match(marker.re);
    if (match && match.index !== undefined) {
      if (!best || match.index < (best.match.index ?? Infinity)) {
        best = { marker, match };
      }
    }
  }
  if (!best) return renderLeaf(text);

  const { marker, match } = best;
  const start = match.index!;
  const before = text.slice(0, start);
  const inner = match[1];
  const after = text.slice(start + match[0].length);
  return [
    ...renderLeaf(before),
    marker.wrap(renderInline(inner), keySeq++),
    ...renderInline(after),
  ];
}

// renderRichText renders a message body: fenced code blocks are extracted
// first (their content is verbatim), the rest is inline-formatted.
export function renderRichText(text: string): ReactNode {
  keySeq = 0;
  const parts = text.split(/```/);
  // Odd indices are the inside of ``` fences.
  const nodes: ReactNode[] = parts.map((part, i) => {
    if (i % 2 === 1) {
      return <CodeBlock key={`cb-${i}`} code={part.replace(/^\n/, "").replace(/\n$/, "")} />;
    }
    return <Fragment key={`t-${i}`}>{renderInline(part)}</Fragment>;
  });
  return <>{nodes}</>;
}

function CodeBlock({ code }: { code: string }) {
  return (
    <span className="rt-codeblock">
      <button
        className="rt-codeblock__copy"
        title="Копировать"
        onClick={(e) => {
          e.stopPropagation();
          void navigator.clipboard?.writeText(code).catch(() => {});
        }}
      >
        Копировать
      </button>
      <code>{code}</code>
    </span>
  );
}

/** First http(s) URL in a text, for link-preview lookup. */
export function firstUrl(text: string): string | null {
  const m = text.match(URL);
  return m ? m[0] : null;
}
