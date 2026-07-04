import type { ReactNode } from "react";
import { Logo } from "@shared/ui";
import "./auth.css";

interface Props {
  subtitle: string;
  children: ReactNode;
}

export function AuthLayout({ subtitle, children }: Props) {
  return (
    <div className="auth-screen">
      <div className="auth-card glass-surface">
        <div className="auth-brand">
          <Logo size={72} className="auth-logo" />
          <h1 className="auth-title">KISY</h1>
          <p className="auth-subtitle">{subtitle}</p>
        </div>
        {children}
      </div>
    </div>
  );
}
