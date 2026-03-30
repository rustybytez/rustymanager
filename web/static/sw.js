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
    icon: '/static/icon-192.png',
    data: { url: data.url || '/' },
    tag: 'chat-' + (data.url || 'default'),
    renotify: true,
    vibrate: [200, 100, 200],
  };
  var targetUrl = data.url || '/';
  e.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (list) {
      for (var i = 0; i < list.length; i++) {
        if (list[i].url.includes(targetUrl) && list[i].focused) {
          return; // user is already looking at the page
        }
      }
      // Increment app badge on PWA icon
      if (self.navigator.setAppBadge) {
        self.navigator.setAppBadge();
      }
      return self.registration.showNotification(title, options);
    })
  );
});

self.addEventListener('notificationclick', function (e) {
  e.notification.close();
  // Clear app badge when user taps notification
  if (self.navigator.clearAppBadge) {
    self.navigator.clearAppBadge();
  }
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
