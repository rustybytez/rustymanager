// Force new SW to activate immediately, replacing old cached version.
self.addEventListener('install', function () { self.skipWaiting(); });
self.addEventListener('activate', function (e) { e.waitUntil(clients.claim()); });

self.addEventListener('push', function (e) {
  var data = {};
  if (e.data) {
    try { data = e.data.json(); } catch (_) {}
  }
  var title = data.title || 'New message';
  var options = {
    body: data.body || '',
    data: { url: data.url || '/' },
    tag: 'chat-' + (data.url || 'default'),
    renotify: true,
  };
  e.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (list) {
      var anyVisible = list.some(function (c) { return c.visibilityState === 'visible'; });
      if (anyVisible) return;
      return self.registration.showNotification(title, options);
    })
  );
});

self.addEventListener('notificationclick', function (e) {
  e.notification.close();
  var url = e.notification.data && e.notification.data.url ? e.notification.data.url : '/';
  e.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (list) {
      for (var i = 0; i < list.length; i++) {
        if (list[i].url.includes(url)) {
          return list[i].focus();
        }
      }
      return clients.openWindow(url);
    })
  );
});
