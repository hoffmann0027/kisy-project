import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { QueryProvider } from "./providers/QueryProvider";
import { router } from "./router";
import { useAuthStore } from "@shared/store/auth";
import { useThemeStore } from "@shared/store/theme";
import { ToastHost } from "@shared/ui";

export function App() {
  const bootstrap = useAuthStore((s) => s.bootstrap);
  const theme = useThemeStore((s) => s.theme);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  // Reflect the active theme on <html> so theme.css selects the token set.
  // An inline script in index.html sets the initial value before first paint
  // (no flash); this keeps it in sync when the user toggles at runtime.
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  return (
    <QueryProvider>
      <RouterProvider router={router} />
      <ToastHost />
    </QueryProvider>
  );
}
