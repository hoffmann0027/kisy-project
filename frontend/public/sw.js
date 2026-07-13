// Minimal service worker: enough to make KISY installable ("Add to Home
// Screen") and to serve the app shell offline. It deliberately never caches
// API or WebSocket traffic — only the static shell and build assets — so no
// authenticated data is persisted on disk.
// Bump this whenever the shell caching behavior changes: the new bytes make
// browsers install the updated worker on next navigation, which purges the
// old cache in activate and takes control (skipWaiting + clients.claim).
const CACHE = "kisy-shell-v4";
const SHELL = ["/", "/favicon.png", "/manifest.webmanifest", "/icon-192.png", "/icon-512.png"];

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

// Web Push: show the notification pushed by the backend.
self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = {};
  }
  event.waitUntil(
    self.registration.showNotification(data.title || "KISY", {
      body: data.body || "",
      icon: "/icon-192.png",
      badge: "/icon-192.png",
      tag: data.tag || "kisy",
      data: { url: data.url || "/" },
    }),
  );
});

// Focus an existing tab (or open one) on the target URL when clicked.
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = (event.notification.data && event.notification.data.url) || "/";
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
      for (const client of list) {
        if ("focus" in client) {
          if ("navigate" in client) client.navigate(target).catch(() => {});
          return client.focus();
        }
      }
      return self.clients.openWindow(target);
    }),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  // Never touch API or realtime traffic.
  if (url.pathname.startsWith("/api") || url.pathname.startsWith("/ws")) return;

  // App navigations: always fetch the HTML shell fresh from the network
  // (bypassing the HTTP cache), falling back to the cached shell only when
  // offline. Without `cache: "no-store"` the browser could hand back a
  // heuristically-cached index.html that still points at a previous deploy's
  // hashed bundles, so new releases would never take effect.
  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request, { cache: "no-store" }).catch(() =>
        caches.match("/").then((r) => r || caches.match("/index.html")),
      ),
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
