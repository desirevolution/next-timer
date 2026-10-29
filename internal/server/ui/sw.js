// PWA shell v32. All asset URLs are scoped to the service worker directory,
// allowing the same bundle under / or /next-timer/ on any static host.
// Updates are installed in the background and activated only by confirmation.
const VERSION = 'v32';
const BASE = new URL('./', self.location.href);
const CACHE_PREFIX = `nexttimer-pwa-${encodeURIComponent(BASE.pathname)}-`;
const CACHE = CACHE_PREFIX + VERSION;
const ASSETS = [
  './', 'index.html', 'styles.css?v=32', 'js/app.js?v=32',
  'js/favorite-sort.js?v=32', 'js/local-timers.js?v=32',
  'js/standalone-store.js?v=32', 'js/quick-timer.js?v=32', 'js/alerts.js?v=32',
  'pwa-update.js?v=32', 'manifest.webmanifest',
  'icons/icon-192.png', 'icons/icon-512.png', 'icons/icon-alarm.png'
];
const absolute = path => new URL(path, BASE).href;

self.addEventListener('install', event => {
  event.waitUntil((async () => {
    // All-or-nothing. Never cache error pages in place of app assets.
    const entries = await Promise.all(ASSETS.map(async path => {
      const response = await fetch(new Request(absolute(path), {cache:'reload'}));
      const type=response.headers.get('Content-Type')||'';
      const expected=/\.js(?:\?|$)/.test(path) ? /javascript|ecmascript/i :
        /\.css(?:\?|$)/.test(path) ? /text\/css/i :
        /\.png(?:\?|$)/.test(path) ? /image\/png/i :
        /\.webmanifest(?:\?|$)/.test(path) ? /json/i : /text\/html/i;
      if(!response.ok||!expected.test(type))
        throw Error(`Invalid offline asset ${path}: ${response.status}, ${type}`);
      return [absolute(path),response];
    }));
    const cache=await caches.open(CACHE);
    await Promise.all(entries.map(([url,response])=>cache.put(url,response)));
  })());
});

self.addEventListener('message', event => {
  if(event.data?.type==='SKIP_WAITING') self.skipWaiting();
});
self.addEventListener('activate', event => {
  event.waitUntil((async()=>{
    const names=await caches.keys();
    await Promise.all(names.filter(x=>x.startsWith(CACHE_PREFIX)&&x!==CACHE).map(x=>caches.delete(x)));
    // Root installs upgrade legacy v23 caches. Don't delete unrelated scopes.
    if(BASE.pathname==='/') await Promise.all(names.filter(x=>/^nexttimer-pwa-v\d+$/.test(x)).map(x=>caches.delete(x)));
    await self.clients.claim();
  })());
});
self.addEventListener('fetch', event => {
  const req=event.request;
  if(req.method!=='GET') return;
  const url=new URL(req.url);
  if(url.origin!==BASE.origin||!url.pathname.startsWith(BASE.pathname))return;
  if(url.pathname.endsWith('/sw.js') || url.pathname.endsWith('/pwa-repair.html') ||
    url.pathname.slice(BASE.pathname.length).startsWith('api/'))return;
  if(req.mode==='navigate') {
    event.respondWith((async()=>{
      try {
        const response=await fetch(req,{cache:'no-store'});
        if(response.status<500) return response;
      } catch { /* offline */ }
      const cache=await caches.open(CACHE);
      return await cache.match(absolute('index.html')) || await cache.match(absolute('./')) ||
        new Response('NextTimer is offline and its app shell is unavailable.',
          {status:503,headers:{'Content-Type':'text/plain; charset=utf-8'}});
    })());
    return;
  }
  if(url.pathname.endsWith('/pwa-update.js')) {
    event.respondWith((async()=>{
      try {const response=await fetch(req,{cache:'no-store'});if(response.ok)return response;}catch{}
      return await (await caches.open(CACHE)).match(req)||Response.error();
    })());
    return;
  }
  event.respondWith((async()=>{
    const cache=await caches.open(CACHE);
    const saved=await cache.match(req);
    if(saved)return saved;
    try{return await fetch(req);}catch{return new Response('Offline resource unavailable',{status:503});}
  })());
});
self.addEventListener('notificationclick', event=>{
  event.notification.close();
  event.waitUntil(self.clients.matchAll({type:'window',includeUncontrolled:true}).then(clients=>{
    const win=clients.find(c=>c.url.startsWith(BASE.href));
    return win?win.focus():self.clients.openWindow(BASE.href);
  }));
});
