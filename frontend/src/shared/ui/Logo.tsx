interface Props {
  /** Rendered width/height in px (the logo is square). */
  size?: number;
  className?: string;
}

// Logo is the KISY/SChat app mark used in-app (nav rail, auth screen): a
// circular badge with the orange "S" glyph on the dark tile (public/logo.png),
// matching the round icon buttons. The favicon and PWA icons use the full
// square-tile artwork (public/favicon.png, icon-*.png) instead — those want the
// wordmark and OS-side masking.
export function Logo({ size = 40, className }: Props) {
  return (
    <img
      src="/logo.png?v=3"
      width={size}
      height={size}
      className={className}
      alt="KISY"
      draggable={false}
      style={{ display: "block", objectFit: "contain" }}
    />
  );
}
