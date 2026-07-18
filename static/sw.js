/* Service worker del sitio de campaña.
   Estrategia: network-first para HTML (contenido siempre fresco mientras se
   itera), cache-first para assets estáticos (css inline, js, img, fonts).
   %%VERSION%% lo reemplaza el generador con el hash del contenido. */
var CACHE = "em-campana-%%VERSION%%";
var PRECACHE = [
  "/",
  "/propuesta/",
  "/trayectoria/",
  "/cv/",
  "/galeria/",
  "/contacto/",
  "/js/site.js",
  "/js/search.js",
  "/fonts/fraunces-latin.woff2",
  "/fonts/inter-latin.woff2",
  "/manifest.webmanifest"
];

self.addEventListener("install", function (ev) {
  ev.waitUntil(
    caches.open(CACHE).then(function (cache) {
      return cache.addAll(PRECACHE);
    }).then(function () { return self.skipWaiting(); })
  );
});

self.addEventListener("activate", function (ev) {
  ev.waitUntil(
    caches.keys().then(function (keys) {
      return Promise.all(
        keys.filter(function (k) { return k !== CACHE; })
            .map(function (k) { return caches.delete(k); })
      );
    }).then(function () { return self.clients.claim(); })
  );
});

self.addEventListener("fetch", function (ev) {
  var req = ev.request;
  if (req.method !== "GET" || new URL(req.url).origin !== location.origin) return;

  var isHTML = req.mode === "navigate" ||
    (req.headers.get("accept") || "").indexOf("text/html") >= 0;

  if (isHTML || req.url.indexOf("search-index.json") >= 0) {
    // Red primero; si falla (offline), servir de caché.
    ev.respondWith(
      fetch(req).then(function (res) {
        var copy = res.clone();
        caches.open(CACHE).then(function (c) { c.put(req, copy); });
        return res;
      }).catch(function () {
        return caches.match(req).then(function (hit) {
          return hit || caches.match("/");
        });
      })
    );
    return;
  }

  // Assets: caché primero, con relleno desde red.
  ev.respondWith(
    caches.match(req).then(function (hit) {
      return hit || fetch(req).then(function (res) {
        var copy = res.clone();
        caches.open(CACHE).then(function (c) { c.put(req, copy); });
        return res;
      });
    })
  );
});
