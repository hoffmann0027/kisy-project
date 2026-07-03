import type { ButtonHTMLAttributes } from "react";
import { cn } from "@shared/lib/cn";
import { Spinner } from "./Spinner";

type Variant = "primary" | "secondary" | "ghost" | "danger";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  block?: boolean;
  loading?: boolean;
}

export function Button({ variant = "primary", block, loading, disabled, children, className, ...rest }: Props) {
  return (
    <button
      className={cn("ui-btn", `ui-btn--${variant}`, block && "ui-btn--block", className)}
      disabled={disabled || loading}
      {...rest}
    >
      {loading && <Spinner size={16} />}
      {children}
    </button>
  );
}
