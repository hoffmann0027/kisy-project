import type { ButtonHTMLAttributes } from "react";
import { cn } from "@shared/lib/cn";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
}

export function IconButton({ label, className, children, ...rest }: Props) {
  return (
    <button type="button" className={cn("ui-icon-btn", className)} aria-label={label} title={label} {...rest}>
      {children}
    </button>
  );
}
