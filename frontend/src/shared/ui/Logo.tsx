interface Props {
  /** Rendered width/height in px (the logo is square). */
  size?: number;
  className?: string;
}

// Logo is the KISY app mark: an orange rounded-tile icon with the stylised
// "K" cat and KISY wordmark. The same artwork backs public/favicon.png and the
// PWA icons, so the header, tab icon and installed-app icon all match. It is a
// raster asset (public/logo.png) served with transparent corners so it sits
// cleanly on any theme background.
export function Logo({ size = 40, className }: Props) {
  return (
    <img
      src="/logo.png"
      width={size}
      height={size}
      className={className}
      alt="KISY"
      draggable={false}
      style={{ display: "block", objectFit: "contain" }}
    />
  );
}
