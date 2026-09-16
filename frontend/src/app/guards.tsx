import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useAuthStore } from "@shared/store/auth";
import { capabilitiesOf } from "@shared/lib/useCapabilities";
import { Spinner } from "@shared/ui";
import { ForcePasswordChange } from "@features/auth/ForcePasswordChange";
import { ForceDisplayNameChange } from "@features/auth/ForceDisplayNameChange";
import { OfflineNotice } from "./OfflineNotice";

function FullScreenLoader() {
  return (
    <div style={{ height: "100%", display: "flex", alignItems: "center", justifyContent: "center" }}>
      <Spinner size={32} />
    </div>
  );
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const mustChange = useAuthStore((s) => s.user?.mustChangePassword ?? false);
  const mustRename = useAuthStore((s) => s.user?.displayNameNeedsChange ?? false);
  if (status === "loading") return <FullScreenLoader />;
  // Unreachable is not signed out: never answer a missing network with a
  // password form.
  if (status === "offline") return <OfflineNotice />;
  if (status === "anonymous") return <Navigate to="/login" replace />;
  // A seeded/reset password must be replaced before anything else loads.
  if (mustChange) return <ForcePasswordChange />;
  // Then a name that stopped being allowed (migration 46): a password first,
  // because it guards the account the name belongs to.
  if (mustRename) return <ForceDisplayNameChange />;
  return <>{children}</>;
}

export function RequireCEO({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const user = useAuthStore((s) => s.user);
  if (status === "loading") return <FullScreenLoader />;
  // Unreachable is not signed out: never answer a missing network with a
  // password form.
  if (status === "offline") return <OfflineNotice />;
  if (status === "anonymous") return <Navigate to="/login" replace />;
  if (!capabilitiesOf(user).canAdmin) return <Navigate to="/" replace />;
  return <>{children}</>;
}

// The rating board and the admin panel are both hierarchy screens: the rule
// for who may open them lives in useCapabilities, so the guard, the rail and
// the tab bar cannot drift apart.
// Rating (clan board) is open to clearance levels 1–9. Level 10 ("not in a
// clan") is bounced to the messenger; the rail shows a popup on click, this
// guards direct URL navigation.
export function RequireRatingAccess({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const user = useAuthStore((s) => s.user);
  if (status === "loading") return <FullScreenLoader />;
  // Unreachable is not signed out: never answer a missing network with a
  // password form.
  if (status === "offline") return <OfflineNotice />;
  if (status === "anonymous") return <Navigate to="/login" replace />;
  if (!capabilitiesOf(user).canSeeRating) return <Navigate to="/" replace />;
  return <>{children}</>;
}

export function RedirectIfAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  if (status === "loading") return <FullScreenLoader />;
  // A password form with no way to reach the server only wastes the attempt.
  if (status === "offline") return <OfflineNotice />;
  if (status === "authenticated") return <Navigate to="/" replace />;
  return <>{children}</>;
}
