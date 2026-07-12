import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useAuthStore } from "@shared/store/auth";
import { Spinner } from "@shared/ui";

function FullScreenLoader() {
  return (
    <div style={{ height: "100%", display: "flex", alignItems: "center", justifyContent: "center" }}>
      <Spinner size={32} />
    </div>
  );
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  if (status === "loading") return <FullScreenLoader />;
  if (status === "anonymous") return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export function RequireCEO({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const user = useAuthStore((s) => s.user);
  if (status === "loading") return <FullScreenLoader />;
  if (status === "anonymous") return <Navigate to="/login" replace />;
  if (user?.roleLevel !== 1) return <Navigate to="/" replace />;
  return <>{children}</>;
}

// Rating (clan board) is open to clearance levels 1–9. Level 10 ("not in a
// clan") is bounced to the messenger; the rail shows a popup on click, this
// guards direct URL navigation.
export function RequireRatingAccess({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const user = useAuthStore((s) => s.user);
  if (status === "loading") return <FullScreenLoader />;
  if (status === "anonymous") return <Navigate to="/login" replace />;
  if ((user?.roleLevel ?? 99) > 9) return <Navigate to="/" replace />;
  return <>{children}</>;
}

export function RedirectIfAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  if (status === "loading") return <FullScreenLoader />;
  if (status === "authenticated") return <Navigate to="/" replace />;
  return <>{children}</>;
}
