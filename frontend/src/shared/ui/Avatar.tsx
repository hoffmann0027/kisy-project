import { colorFromString, initials } from "@shared/lib/format";

interface Props {
  name: string;
  url?: string | null;
  size?: number;
  presence?: "online" | "offline" | "away";
}

export function Avatar({ name, url, size = 44, presence }: Props) {
  return (
    <span
      className="ui-avatar"
      style={{
        width: size,
        height: size,
        fontSize: size * 0.4,
        background: url ? undefined : colorFromString(name),
      }}
    >
      {url ? <img src={url} alt={name} /> : initials(name)}
      {presence === "online" && <span className="ui-avatar__presence" />}
    </span>
  );
}
