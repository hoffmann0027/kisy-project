import { cn } from "@shared/lib/cn";
import { useToastStore } from "./toast";

export function ToastHost() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);

  return (
    <div className="ui-toasts">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={cn("ui-toast", t.kind === "error" && "ui-toast--error", t.kind === "success" && "ui-toast--success")}
          onClick={() => dismiss(t.id)}
          role="status"
        >
          {t.message}
        </div>
      ))}
    </div>
  );
}
