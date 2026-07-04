import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "@app/App";
import "@shared/config/theme.css";
import "@shared/ui/ui.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

// Register the service worker in production only, so it never interferes with
// Vite's dev server / HMR. Enables installability and offline app shell.
if (import.meta.env.PROD && "serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {
      // A failed SW registration must not break the app.
    });
  });
}
