import { Fragment, type ReactNode } from "react";

// A translated sentence with markup inside ("Токен действует <strong>ровно
// 120 секунд</strong> и…") stays one message, so a translator can move the
// emphasised part where the language wants it: the message carries a
// `{name}` placeholder that t() leaves alone, and this puts the element there.
export function interpolate(text: string, parts: Record<string, ReactNode>): ReactNode {
  return text.split(/(\{\w+\})/).map((piece, i) => {
    const name = /^\{(\w+)\}$/.exec(piece)?.[1];
    return <Fragment key={i}>{name !== undefined && name in parts ? parts[name] : piece}</Fragment>;
  });
}
