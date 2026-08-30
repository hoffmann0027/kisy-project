import type { ImgHTMLAttributes } from "react";
import { useMediaSrc } from "@shared/lib/mediaSrc";

interface Props extends Omit<ImgHTMLAttributes<HTMLImageElement>, "src"> {
  /** URL as the API reports it, e.g. "/api/v1/attachments/<id>". */
  src: string;
  alt: string;
}

/**
 * An <img> for pictures the API serves behind authentication.
 *
 * On the web it is a plain <img>. In the mobile shell the bytes have to be
 * fetched with the session's bearer token first (see mediaSrc) — a bare tag
 * there resolves the relative URL against the WebView's own origin and sends
 * no credentials, which is why attachments used to show as nothing but their
 * file name on the phone.
 *
 * It exists as a component rather than a hook call at each site because the
 * attachments are rendered inside .map() — a hook cannot live in a loop.
 */
export function ApiImage({ src, alt, ...rest }: Props) {
  const resolved = useMediaSrc(src);
  if (!resolved) return null;
  return <img src={resolved} alt={alt} {...rest} />;
}
