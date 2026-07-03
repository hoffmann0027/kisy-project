import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

export default defineConfig(({ mode }) => {
  // Dev-only proxy target for the backend. Defaults to :8080; override via
  // VITE_PROXY_TARGET in a local .env file when that port is taken (e.g.
  // this machine publishes the backend on :18080 because BlueStacks holds
  // :8080). Does not affect the production build (served by Nginx).
  const env = loadEnv(mode, process.cwd(), "");
  const backendTarget = env.VITE_PROXY_TARGET || "http://localhost:8080";
  const wsTarget = backendTarget.replace(/^http/, "ws");

  return {
    plugins: [react()],
    resolve: {
      alias: {
        "@app": path.resolve(__dirname, "src/app"),
        "@pages": path.resolve(__dirname, "src/pages"),
        "@widgets": path.resolve(__dirname, "src/widgets"),
        "@features": path.resolve(__dirname, "src/features"),
        "@entities": path.resolve(__dirname, "src/entities"),
        "@shared": path.resolve(__dirname, "src/shared"),
      },
    },
    server: {
      host: true,
      port: 5173,
      proxy: {
        // changeOrigin is intentionally left false so the browser's Host
        // header is preserved end-to-end; the backend CSRF check compares
        // Origin against Host, which must therefore stay consistent.
        "/api": { target: backendTarget },
        "/ws": { target: wsTarget, ws: true },
      },
    },
  };
});
