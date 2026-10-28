// A new worker is installed in the background but waits for an explicit update click.
// Bump VERSION AND the asset query in index.html/app.js for each release.
const VERSION = 'v22';
const CACHE = 'nexttimer-pwa-' + VERSION;
const ASSETS = [
  '/', '/index.html', '/styles.css?v=22', '/js/app.js?v=22',
  '/js/favorite-sort.js?v=22', '/js/local-timers.js?v=22', '/js/alerts.js?v=22',
  '/manifest.webmanifest', '/icons/icon-192.png', '/icons/icon-512.png'
];
self.addEventListener('install', event => {
  event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(ASSETS)));
  // No skipWaiting here: older clients must explicitly accept an update.
});
self.addEventListener('message', event => {
  if (event.data?.type === 'SKIP_WAITING') self.skipWaiting();
});
self.addEventListener('activate', event => {
  event.waitUntil(Promise.all([
    caches.keys().then(keys => Promise.all(keys.filter(k => k.startsWith('nexttimer-pwa-') && k !== CACHE).map(k => caches.delete(k)))),
    self.clients.claim()
  ]));
});
self.addEventListener('fetch', event => {
  const req = event.request, url = new URL(req.url);
  if (req.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return;
  // The update coordinator should be re-fetched and never stuck in the app cache.
  if (url.pathname === '/pwa-update.js' || url.pathname === '/sw.js') return;
  if (req.mode === 'navigate') {
    // Keep using the current version until the user accepts the waiting worker.
    event.respondWith(caches.match('/index.html').then(cached => cached || fetch(req)));
    return;
  }
  event.respondWith(caches.match(req).then(cached => cached || fetch(req)));
});
self.addEventListener('notificationclick', event => {
  event.notification.close();
  event.waitUntil(self.clients.matchAll({type:'window', includeUncontrolled:true}).then(clients => {
    const win = clients.find(c => new URL(c.url).origin === self.location.origin);
    return win ? win.focus() : self.clients.openWindow('/');
  }));
});
