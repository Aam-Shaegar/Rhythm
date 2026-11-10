// Build stamps the shell version; hashed backgrounds are cached only when first used.
const CACHE = 'rhythm-dev';
const PRECACHE = [];
self.addEventListener('install', event => {
 event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(PRECACHE)).then(() => self.skipWaiting()));
});
self.addEventListener('activate', event => {
 event.waitUntil(caches.keys().then(keys => Promise.all(keys.filter(k => k.startsWith('rhythm-') && k !== CACHE).map(k => caches.delete(k)))).then(() => self.clients.claim()));
});
self.addEventListener('fetch', event => {
 const url = new URL(event.request.url);
 if (event.request.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api/') || url.pathname === '/config.js') return;
 if (event.request.mode === 'navigate') {
  event.respondWith((async () => {
   const cache = await caches.open(CACHE);
   // Fall back to the installed shell promptly when the network is slow or absent.
   try {
    const res = await fetch(event.request, { signal: AbortSignal.timeout(3500) });
    if (!res.ok) throw new Error('navigation failed');
    return res;
   } catch {
    return await cache.match('/index.html') || Response.error();
   }
  })());
 } else if (url.pathname.startsWith('/assets/') || url.pathname.startsWith('/icons/')) {
  event.respondWith((async () => {
   const cache = await caches.open(CACHE);
   const cached = await cache.match(event.request);
   if (cached) return cached;
   const res = await fetch(event.request);
   if (res.ok) await cache.put(event.request, res.clone());
   return res;
  })());
 }
});

self.addEventListener('push', (event) => {
  let data = { title: 'Ритм', body: 'У вас новое напоминание', url: '/', id: 'reminder' };
  try {
    if (event.data) {
      const parsed = event.data.json();
      data = {
        title: typeof parsed.title === 'string' ? parsed.title : data.title,
        body: typeof parsed.body === 'string' ? parsed.body : data.body,
        id: typeof parsed.id === 'string' ? parsed.id : data.id,
        url: typeof parsed.url === 'string' ? parsed.url : data.url,
      };
    }
  } catch (_) {
    // битый payload — покажем заглушку
  }

  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: '/icons/icon-192.png',
      badge: '/icons/icon-192.png',
      data: { url: data.url },
      tag: `rhythm-${data.id}`,
    }),
  );
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = (event.notification.data && event.notification.data.url) || '/';
  event.waitUntil(
    (async () => {
      const all = await clients.matchAll({ type: 'window', includeUncontrolled: true });
      for (const client of all) {
        if ('focus' in client) {
          if ('navigate' in client) await client.navigate(url);
          await client.focus();
          return;
        }
      }
      if (clients.openWindow) {
        await clients.openWindow(url);
      }
    })(),
  );
});
