import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { QueryProvider } from "./providers/QueryProvider";
import { router } from "./router";
import { useAuthStore } from "@shared/store/auth";
import { useThemeStore } from "@shared/store/theme";
import { useVisualViewport } from "@shared/lib/useVisualViewport";
import { initNativePushNavigation, refreshNativePushToken } from "@shared/lib/nativePush";
import { ToastHost } from "@shared/ui";

export function App() {
  const bootstrap = useAuthStore((s) => s.bootstrap);
  const userId = useAuthStore((s) => s.user?.id);
  const theme = useThemeStore((s) => s.theme);
  const keyboardInset = useVisualViewport();

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  // Mobile shell: opening a notification should land on the chat it came from.
  // Registered once, outside the router, because the tap can arrive while the
  // app is cold-starting.
  useEffect(() => {
    initNativePushNavigation((path) => void router.navigate(path));
  }, []);

  // Firebase tokens rotate and sign-out unbinds the device, so re-register on
  // every sign-in. A no-op in the browser and for anyone who never opted in.
  useEffect(() => {
    if (userId) void refreshNativePushToken();
  }, [userId]);

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
