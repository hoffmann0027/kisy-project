// Minimal service worker: enough to make KISY installable ("Add to Home
// Screen") and to serve the app shell offline. It deliberately never caches
// API or WebSocket traffic — only the static shell and build assets — so no
// authenticated data is persisted on disk.
// Bump this whenever the shell caching behavior changes: the new bytes make
// browsers install the updated worker on next navigation, which purges the
// old cache in activate and takes control (skipWaiting + clients.claim).
const CACHE = "kisy-shell-v7";
const SHELL = ["/", "/favicon.png?v=2", "/manifest.webmanifest", "/icon-192.png?v=2", "/icon-512.png?v=2"];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys();
      const stale = keys.filter((k) => k !== CACHE);
      await Promise.all(stale.map((k) => caches.delete(k)));
      await self.clients.claim();
      // Self-heal a stuck deploy: when this is an UPDATE (an older cache
      // existed), force every open tab to reload through the new worker so a
      // redeploy (new bundle, logo, theme) applies without the user manually
      // clearing the service worker. First installs (no older cache) are left
      // alone so new visitors don't get a spurious reload.
      if (stale.length > 0) {
        const clients = await self.clients.matchAll({ type: "window" });
        for (const client of clients) {
          client.navigate(client.url).catch(() => {});
        }
      }
    })(),
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
      icon: "/icon-192.png?v=2",
      badge: "/icon-192.png?v=2",
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

  // Build assets under /assets/ are content-hashed (index-<hash>.js/.css), so
  // cache-first is safe and fast — a changed file gets a new URL.
  if (url.pathname.startsWith("/assets/")) {
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
    return;
  }

  // Other static files (favicon.png, logo.png, icon-*.png, manifest, fonts)
  // have STABLE names, so cache-first would pin a stale copy forever (this is
  // why an updated logo did not appear). Use network-first: serve the fresh
  // bytes when online, fall back to cache offline, and refresh the cache on
  // every successful fetch.
  if (/\.(png|svg|webmanifest|woff2?)$/.test(url.pathname)) {
    event.respondWith(
      fetch(request)
        .then((resp) => {
          const copy = resp.clone();
          caches.open(CACHE).then((c) => c.put(request, copy));
          return resp;
        })
        .catch(() => caches.match(request)),
    );
  }
});
