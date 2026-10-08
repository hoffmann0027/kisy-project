interface Props {
  count: number;
  /** Read out instead of the bare number ("3 unread"). */
  label: string;
}

/**
 * A red count that asks to be looked at — unread notifications. Placed in the
 * top-right corner of its (positioned) parent; renders nothing at zero.
 */
export function AlertBadge({ count, label }: Props) {
  if (count <= 0) return null;
  return (
    <span className="alert-badge" role="status" aria-label={label}>
      {count > 9 ? "9+" : count}
    </span>
  );
}
