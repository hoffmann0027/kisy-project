// Minimal service worker: enough to make KISY installable ("Add to Home
// Screen") and to serve the app shell offline. It deliberately never caches
// API or WebSocket traffic — only the static shell and build assets — so no
// authenticated data is persisted on disk.
const CACHE = "kisy-shell-v1";
const SHELL = ["/", "/favicon.svg", "/manifest.webmanifest", "/icon-192.png", "/icon-512.png"];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  // Never touch API or realtime traffic.
  if (url.pathname.startsWith("/api") || url.pathname.startsWith("/ws")) return;

  // App navigations: network-first, fall back to the cached shell offline.
  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request).catch(() => caches.match("/").then((r) => r || caches.match("/index.html"))),
    );
    return;
  }

  // Build assets and icons: cache-first (they are content-hashed / versioned).
  if (url.pathname.startsWith("/assets/") || /\.(png|svg|webmanifest|woff2?)$/.test(url.pathname)) {
    event.respondWith(
      caches.match(request).then(
        (hit) =>
          hit ||
          fetch(request).then((resp) => {
            const copy = resp.clone();
            caches.open(CACHE).then((c) => c.put(request, copy));
            return resp;
          }),
      ),
    );
  }
});
