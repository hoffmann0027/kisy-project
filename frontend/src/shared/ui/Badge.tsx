import type { ReactNode } from "react";
import { cn } from "@shared/lib/cn";

interface Props {
  children: ReactNode;
  muted?: boolean;
}

export function Badge({ children, muted }: Props) {
  return <span className={cn("ui-badge", muted && "ui-badge--muted")}>{children}</span>;
}
