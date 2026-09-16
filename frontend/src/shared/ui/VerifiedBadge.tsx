import "./VerifiedBadge.css";

// The verification mark: an octagonal seal with a check.
//
// Chosen over a scalloped rosette on purpose — that silhouette belongs to other
// platforms. Faces, not teeth: it reads as an official seal without borrowing
// anyone's. Drawn from theme tokens only (--acc fills it, --acc-on draws the
// check), so it is at home in all seven themes, light and dark.
//
// Two sizes. At 20px it carries an inner rim, like an impressed seal; at 14px
// that rim would blur into the fill, so the small mark is the plain seal with
// a heavier check.

const BODY =
  "M8.9 1.8h6.2c.8 0 1.6.3 2.1.9l4.1 4.1c.6.6.9 1.3.9 2.1v6.2c0 .8-.3 1.6-.9 2.1l-4.1 4.1c-.6.6-1.3.9-2.1.9H8.9c-.8 0-1.6-.3-2.1-.9l-4.1-4.1c-.6-.6-.9-1.3-.9-2.1V8.9c0-.8.3-1.6.9-2.1l4.1-4.1c.6-.6 1.3-.9 2.1-.9Z";
const RIM =
  "M9.6 4.3h4.8c.5 0 .9.2 1.3.5l3.5 3.5c.3.4.5.8.5 1.3v4.8c0 .5-.2.9-.5 1.3l-3.5 3.5c-.4.3-.8.5-1.3.5H9.6c-.5 0-.9-.2-1.3-.5l-3.5-3.5c-.3-.4-.5-.8-.5-1.3V9.6c0-.5.2-.9.5-1.3l3.5-3.5c.4-.3.8-.5 1.3-.5Z";

export type VerifiedBadgeSize = 14 | 20;

interface Props {
  size?: VerifiedBadgeSize;
  /** What is verified, for the accessible label: a person or a community. */
  subject?: "user" | "group";
  className?: string;
}

export function VerifiedBadge({ size = 14, subject = "user", className }: Props) {
  const detailed = size >= 20;
  const label = subject === "group" ? "Подтверждённое сообщество" : "Подтверждённый аккаунт";
  return (
    <svg
      className={["verified-badge", className].filter(Boolean).join(" ")}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      role="img"
      aria-label={label}
    >
      <title>{label}</title>
      <path fill="var(--acc)" d={BODY} />
      {detailed && <path fill="none" stroke="var(--acc-on)" strokeOpacity={0.55} strokeWidth={1} d={RIM} />}
      <path
        fill="none"
        stroke="var(--acc-on)"
        strokeWidth={detailed ? 2.1 : 2.8}
        strokeLinecap="round"
        strokeLinejoin="round"
        d={detailed ? "m8.6 12.1 2.4 2.4 4.6-4.8" : "m8 12.3 2.8 2.8 5.2-5.6"}
      />
    </svg>
  );
}

interface NameProps {
  name: string;
  verified: boolean;
  size?: VerifiedBadgeSize;
  subject?: "user" | "group";
  className?: string;
}

/**
 * A name with its mark beside it. The name truncates, the mark never does: a
 * badge clipped by an ellipsis would be the one part of the line that carries
 * meaning and cannot be read.
 */
export function VerifiedName({ name, verified, size = 14, subject = "user", className }: NameProps) {
  return (
    <span className={["verified-name", className].filter(Boolean).join(" ")}>
      <span className="verified-name__text">{name}</span>
      {verified && <VerifiedBadge size={size} subject={subject} />}
    </span>
  );
}
