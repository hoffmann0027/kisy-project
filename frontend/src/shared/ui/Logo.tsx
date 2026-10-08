interface Props {
  /** Rendered width/height in px (the logo is square). */
  size?: number;
  className?: string;
}

// Logo is the KISY app mark used in-app (nav rail, auth screen): the orange
// "K" bubble in a clear glass tile (public/logo.png) — transparent, so each of
// the seven themes shows through the glass. The favicon and PWA icons are the
// same tile. All of them are generated from design/logo-source.png; bump the
// ?v= below (and in index.html, manifest.webmanifest, sw.js) when the artwork
// changes.
export function Logo({ size = 40, className }: Props) {
  return (
    <img
      src="/logo.png?v=7"
      width={size}
      height={size}
      className={className}
      alt="KISY"
      draggable={false}
      style={{ display: "block", objectFit: "contain" }}
    />
  );
}
