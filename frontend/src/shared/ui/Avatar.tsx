import { colorFromString, initials } from "@shared/lib/format";
import { useMediaSrc } from "@shared/lib/mediaSrc";

interface Props {
  name: string;
  url?: string | null;
  size?: number;
  presence?: "online" | "offline" | "away";
}

export function Avatar({ name, url, size = 44, presence }: Props) {
  // Avatars are served by the API behind auth: inside the mobile shell an
  // <img> cannot reach them on its own (see mediaSrc).
  const src = useMediaSrc(url);
  return (
    <span
      className="ui-avatar"
      style={{
        width: size,
        height: size,
        fontSize: size * 0.4,
        background: src ? undefined : colorFromString(name),
      }}
    >
      {src ? <img src={src} alt={name} /> : initials(name)}
      {presence === "online" && <span className="ui-avatar__presence" />}
    </span>
  );
}
