interface Props {
  /** Rendered width/height in px (the logo is square). */
  size?: number;
  className?: string;
}

// Logo is the KISY app mark used in-app (nav rail, auth screen): the orange
// "K" bubble on a transparent background (public/logo.png), so it sits
// cleanly on all seven themes. The favicon and PWA icons use the same mark on
// its dark tile (public/favicon.png, icon-*.png) instead — those sit on OS or
// browser chrome and want an opaque ground. All of them are generated from
// design/logo-source.png; bump the ?v= below when the artwork changes.
export function Logo({ size = 40, className }: Props) {
  return (
    <img
      src="/logo.png?v=5"
      width={size}
      height={size}
      className={className}
      alt="KISY"
      draggable={false}
      style={{ display: "block", objectFit: "contain" }}
    />
  );
}
