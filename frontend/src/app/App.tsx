import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { QueryProvider } from "./providers/QueryProvider";
import { router } from "./router";
import { useAuthStore } from "@shared/store/auth";
import { ToastHost } from "@shared/ui";

export function App() {
  const bootstrap = useAuthStore((s) => s.bootstrap);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  return (
    <QueryProvider>
      <RouterProvider router={router} />
      <ToastHost />
    </QueryProvider>
  );
}
