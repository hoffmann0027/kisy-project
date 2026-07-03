import { useQuery } from "@tanstack/react-query";
import { Spinner } from "@shared/ui";
import { adminApi } from "@shared/api/endpoints";

export function AuditTab() {
  const { data, isPending } = useQuery({
    queryKey: ["admin", "audit"],
    queryFn: async () => (await adminApi.audit()).entries,
  });

  if (isPending) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 40 }}>
        <Spinner size={28} />
      </div>
    );
  }

  return (
    <table className="table">
      <thead>
        <tr>
          <th>Время</th>
          <th>Действие</th>
          <th>Объект</th>
          <th>Детали</th>
        </tr>
      </thead>
      <tbody>
        {data?.map((e) => (
          <tr key={e.id}>
            <td style={{ whiteSpace: "nowrap", color: "var(--color-text-secondary)" }}>
              {new Date(e.createdAt).toLocaleString("ru-RU")}
            </td>
            <td>
              <span className="audit-action">{e.action}</span>
            </td>
            <td style={{ color: "var(--color-text-secondary)" }}>{e.targetType ?? "—"}</td>
            <td style={{ fontSize: 12, color: "var(--color-text-tertiary)", maxWidth: 320 }}>
              {Object.keys(e.metadata).length > 0 ? JSON.stringify(e.metadata) : "—"}
            </td>
          </tr>
        ))}
        {(data?.length ?? 0) === 0 && (
          <tr>
            <td colSpan={4} style={{ textAlign: "center", color: "var(--color-text-secondary)", padding: 24 }}>
              Записи аудита отсутствуют
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
