import "@shared/config/theme.css";
import "@shared/ui/ui.css";
import { initI18n } from "@shared/i18n";

// Choose the language and load its words first, then start the app: no screen
// ever renders in one language and flips to another.
void initI18n().then(() => import("./bootstrap"));

// Register the service worker in production only, so it never interferes with
// Vite's dev server / HMR. Enables installability and offline app shell.
// `updateViaCache: "none"` makes the browser always revalidate sw.js itself
// (never serve the worker script from the HTTP cache), so a redeploy is
// detected promptly; the worker's activate step then refreshes open tabs.
if (import.meta.env.PROD && "serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker
      .register("/sw.js", { updateViaCache: "none" })
      .then((reg) => reg.update())
      .catch(() => {
        // A failed SW registration must not break the app.
      });
  });
}
