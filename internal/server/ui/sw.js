// PWA shell v23. Keep the last successfully installed bundle for offline use.
// Update assets and VERSION together. Updates activate only after the user agrees.
const VERSION = 'v23';
const CACHE = 'nexttimer-pwa-' + VERSION;
const ASSETS = [
  '/', '/index.html', '/styles.css?v=23', '/js/app.js?v=23',
  '/js/favorite-sort.js?v=23', '/js/local-timers.js?v=23', '/js/alerts.js?v=23',
  '/manifest.webmanifest', '/icons/icon-192.png', '/icons/icon-512.png'
];

self.addEventListener('install', event => {
  // Fetch the entire bundle before writing it to Cache Storage. Verify the
  // content types: a login page or proxy error returned as HTTP 200 must not
  // silently be cached as JavaScript/CSS and break the installed PWA.
  event.waitUntil((async () => {
    const entries = await Promise.all(ASSETS.map(async path => {
      const response = await fetch(new Request(path, {cache: 'reload'}));
      const type = response.headers.get('Content-Type') || '';
      const expected = path.includes('.js') ? /javascript|ecmascript/i :
        path.includes('.css') ? /text\/css/i :
        path.includes('.png') ? /image\/png/i :
        path.includes('.webmanifest') ? /json/i : /text\/html/i;
      if (!response.ok || !expected.test(type)) {
        throw new Error('Invalid offline asset: ' + path + ' (' + response.status + ', ' + type + ')');
      }
      return [path, response];
    }));
    const cache = await caches.open(CACHE);
    await Promise.all(entries.map(([path, response]) => cache.put(path, response)));
  })());
});

self.addEventListener('message', event => {
  if (event.data?.type === 'SKIP_WAITING') self.skipWaiting();
});

self.addEventListener('activate', event => {
  event.waitUntil((async () => {
    const names = await caches.keys();
    await Promise.all(names.filter(name => name.startsWith('nexttimer-pwa-') && name !== CACHE).map(name => caches.delete(name)));
    await self.clients.claim();
  })());
});

self.addEventListener('fetch', event => {
  const req = event.request;
  if (req.method !== 'GET') return;
  const url = new URL(req.url);
  if (url.origin !== self.location.origin ||
      url.pathname.startsWith('/api/') ||
      url.pathname === '/sw.js' || url.pathname === '/pwa-update.js') return;

  if (req.mode === 'navigate') {
    // Never pin the user to a stale HTML document while online. The offline
    // fallback stays tied to the installed worker version and its matching JS.
    event.respondWith((async () => {
      try {
        const response = await fetch(req, {cache: 'no-store'});
        if (response.status < 500) return response;
      } catch { /* Offline: load the last known-good shell below. */ }
      const cache = await caches.open(CACHE);
      const fallback = await cache.match('/index.html') || await cache.match('/');
      return fallback || new Response('NextTimer is offline and no cached app is available.', {
        status: 503, headers: {'Content-Type': 'text/plain; charset=utf-8'}
      });
    })());
    return;
  }

  // Only use assets from the current installed bundle. Avoid caches.match()
  // across versions, which could serve mismatched JS/CSS after an upgrade.
  event.respondWith((async () => {
    const cache = await caches.open(CACHE);
    const cached = await cache.match(req);
    if (cached) return cached;
    try {
      return await fetch(req);
    } catch {
      return new Response('Offline resource unavailable', {
        status: 503, headers: {'Content-Type': 'text/plain; charset=utf-8'}
      });
    }
  })());
});

self.addEventListener('notificationclick', event => {
  event.notification.close();
  event.waitUntil(self.clients.matchAll({type: 'window', includeUncontrolled: true}).then(clients => {
    const win = clients.find(c => new URL(c.url).origin === self.location.origin);
    return win ? win.focus() : self.clients.openWindow('/');
  }));
});
