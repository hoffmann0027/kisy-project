interface Props {
  /** Rendered width/height in px (the logo is square). */
  size?: number;
  className?: string;
}

// Logo is the KISY app mark: a gradient speech-bubble frame around the KISY
// wordmark, drawn entirely as vector strokes so it stays crisp at any size and
// needs no web font. The same artwork backs public/favicon.svg and the PWA
// icons, so the header, tab icon and installed-app icon all match.
export function Logo({ size = 40, className }: Props) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 512 512"
      className={className}
      role="img"
      aria-label="KISY"
    >
      <defs>
        <linearGradient id="kisyGradInline" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#33c6f6" />
          <stop offset="0.55" stopColor="#5a8bf6" />
          <stop offset="1" stopColor="#7b34f2" />
        </linearGradient>
      </defs>
      <rect x="24" y="24" width="464" height="464" rx="116" fill="#0a0a0c" />
      <rect x="24.5" y="24.5" width="463" height="463" rx="115.5" fill="none" stroke="#ffffff" strokeOpacity="0.06" />

      <g fill="none" stroke="url(#kisyGradInline)" strokeWidth="20" strokeLinecap="round" strokeLinejoin="round">
        <path d="M132 178 C132 141 151 128 187 128 L325 128 C361 128 380 141 380 178" />
        <path d="M132 334 C132 371 151 384 187 384 L232 384 L248 414 L266 384 L325 384 C361 384 380 371 380 334" />
      </g>

      <g transform="translate(-4 0)" fill="none" stroke="#ffffff" strokeWidth="24" strokeLinecap="round" strokeLinejoin="round">
        <path d="M150 210 L150 302" />
        <path d="M150 258 L196 210" />
        <path d="M150 256 L200 302" />
        <path d="M232 210 L232 302" />
        <path d="M316 226 C302 213 272 213 272 236 C272 255 316 259 316 280 C316 303 286 303 270 289" />
        <path d="M334 210 L356 252" />
        <path d="M378 210 L356 252" />
        <path d="M356 252 L356 302" />
      </g>
      <circle cx="356" cy="196" r="10" fill="#ffffff" />
    </svg>
  );
}
