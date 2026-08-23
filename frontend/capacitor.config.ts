import type { CapacitorConfig } from "@capacitor/cli";

// Android/iOS shell around the existing SPA bundle. The WebView serves the
// bundle locally from https://localhost, so the API is cross-origin: the app
// authenticates with a Bearer token (see src/shared/lib/native.ts) and the
// backend allows this origin through NATIVE_APP_ORIGINS.
const config: CapacitorConfig = {
  appId: "com.kisy.messenger",
  appName: "KISY",
  webDir: "dist",
  android: {
    // https (not http) keeps the WebView on a secure origin, which the browser
    // APIs the app relies on require: getUserMedia for voice calls, WebCrypto
    // for the E2EE key material, and service workers.
    androidScheme: "https",
  },
  server: {
    // Cleartext stays off: the app talks to the production origin over TLS.
    androidScheme: "https",
  },
};

export default config;
