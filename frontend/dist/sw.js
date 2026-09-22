/* Ритм — service worker: только Web Push. Без кэширования страниц,
   чтобы свежие сборки прилетали сами ( dist fingerprinted ). */

self.addEventListener('push', (event) => {
  let data = { title: 'Ритм', body: 'У вас новое напоминание', url: '/' };
  try {
    if (event.data) {
      const parsed = event.data.json();
      data = {
        title: typeof parsed.title === 'string' ? parsed.title : data.title,
        body: typeof parsed.body === 'string' ? parsed.body : data.body,
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
      tag: 'rhythm-reminder',
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
