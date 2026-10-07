import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { nativeContentSecurityPolicy } from "./src/shared/config/nativeCsp";

export default defineConfig(({ mode }) => {
  // Dev-only proxy target for the backend. Defaults to :8080; override via
  // VITE_PROXY_TARGET in a local .env file when that port is taken (e.g.
  // this machine publishes the backend on :18080 because BlueStacks holds
  // :8080). Does not affect the production build (served by Nginx).
  const env = loadEnv(mode, process.cwd(), "");
  const backendTarget = env.VITE_PROXY_TARGET || "http://localhost:8080";
  const wsTarget = backendTarget.replace(/^http/, "ws");
  // A native build (VITE_NATIVE_API_ORIGIN set, see the android workflows)
  // has no server to send a CSP header, so the policy goes into index.html.
  const nativeApi = env.VITE_NATIVE_API_ORIGIN;

  return {
    plugins: [
      react(),
      {
        name: "kisy-native-csp",
        apply: "build",
        transformIndexHtml: () =>
          nativeApi
            ? [
                {
                  tag: "meta",
                  attrs: { "http-equiv": "Content-Security-Policy", content: nativeContentSecurityPolicy(nativeApi) },
                  injectTo: "head-prepend",
                },
              ]
            : [],
      },
    ],
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
    build: {
      rollupOptions: {
        output: {
          // Split the heavyweight dependencies out of the entry chunk. They
          // are big, they change rarely, and most of them are not needed to
          // paint the first screen — keeping them separate lets the browser
          // cache them across releases and parse them off the critical path.
          manualChunks: {
            sodium: ["libsodium-wrappers-sumo"],
            mls: ["ts-mls"],
            query: ["@tanstack/react-query"],
          },
        },
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
