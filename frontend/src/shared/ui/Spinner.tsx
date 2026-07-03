interface Props {
  size?: number;
}

export function Spinner({ size = 20 }: Props) {
  return (
    <span
      className="ui-spinner"
      style={{ width: size, height: size, borderWidth: Math.max(2, Math.round(size / 10)) }}
      role="status"
      aria-label="Загрузка"
    />
  );
}
