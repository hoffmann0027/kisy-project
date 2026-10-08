// The app itself. Imported by main.tsx only after the language is chosen, so
// every screen module starts out in it.
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "@app/App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
