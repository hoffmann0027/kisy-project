import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { QueryProvider } from "./providers/QueryProvider";
import { router } from "./router";
import { useAuthStore } from "@shared/store/auth";
import { useThemeStore } from "@shared/store/theme";
import { useVisualViewport } from "@shared/lib/useVisualViewport";
import { ToastHost } from "@shared/ui";

export function App() {
  const bootstrap = useAuthStore((s) => s.bootstrap);
  const theme = useThemeStore((s) => s.theme);
  const keyboardInset = useVisualViewport();

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  // Reflect the active theme on <html> so theme.css selects the token set.
  // An inline script in index.html sets the initial value before first paint
  // (no flash); this keeps it in sync when the user toggles at runtime.
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  // Publish the keyboard overlap as a CSS token so layouts can shrink out of
  // its way (theme.css defaults it to 0px for the pre-hydration paint and for
  // desktop, where visualViewport reports no overlap).
  useEffect(() => {
    document.documentElement.style.setProperty("--kb-inset", `${keyboardInset}px`);
  }, [keyboardInset]);

  return (
    <QueryProvider>
      <RouterProvider router={router} />
      <ToastHost />
    </QueryProvider>
  );
}
