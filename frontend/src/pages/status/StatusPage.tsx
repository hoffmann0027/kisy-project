import { useQuery } from "@tanstack/react-query";

interface HealthPayload {
  status: string;
}

interface HealthEnvelope {
  success: boolean;
  data?: HealthPayload;
}

async function fetchHealth(path: "/health" | "/ready"): Promise<string> {
  const response = await fetch(path);
  const envelope: HealthEnvelope = await response.json();
  if (!envelope.success || !envelope.data) {
    throw new Error(`${path} reported failure`);
  }
  return envelope.data.status;
}

function StatusRow({ label, path }: { label: string; path: "/health" | "/ready" }) {
  const { data, isPending, isError } = useQuery({
    queryKey: ["backend-status", path],
    queryFn: () => fetchHealth(path),
  });

  const state = isPending ? "checking…" : isError ? "unreachable" : data;
  const color = isPending ? "var(--color-text-secondary)" : isError ? "var(--color-danger)" : "var(--color-success)";

  return (
    <div style={{ display: "flex", justifyContent: "space-between", padding: "8px 0" }}>
      <span style={{ color: "var(--color-text-secondary)" }}>{label}</span>
      <span style={{ color, fontWeight: 600 }}>{state}</span>
    </div>
  );
}

export function StatusPage() {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        minHeight: "100%",
        padding: 24,
      }}
    >
      <div className="glass-surface" style={{ width: 360, padding: 32 }}>
        <h1 style={{ margin: 0, fontSize: 28, letterSpacing: -0.5 }}>KISY</h1>
        <p style={{ color: "var(--color-text-secondary)", marginTop: 4, marginBottom: 24 }}>
          Enterprise Messenger — фундамент проекта
        </p>
        <StatusRow label="Backend liveness" path="/health" />
        <StatusRow label="Backend readiness (DB + Redis)" path="/ready" />
      </div>
    </div>
  );
}
