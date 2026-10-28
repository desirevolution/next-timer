// Independent of cached app.js, so an older PWA can discover and offer an update.
// A waiting worker must NEVER replace the running app without a user action.
(() => {
  if (!('serviceWorker' in navigator)) return;
  const $ = id => document.getElementById(id);
  let registration;
  let dismissed = false;
  let userApproved = false;
  let reloadHandled = false;

  function offer() {
    if (!registration?.waiting || !navigator.serviceWorker.controller || dismissed || userApproved) return;
    $('pwa-update-banner').hidden = false;
  }
  function watchInstalling(worker) {
    if (!worker) return;
    worker.addEventListener('statechange', () => {
      if (worker.state === 'installed') offer();
    });
  }
  async function check() {
    if (!navigator.onLine || !registration) return;
    try { await registration.update(); offer(); }
    catch (error) { console.debug('PWA-Update derzeit nicht prüfbar:', error); }
  }
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    // Avoid changing pages while an edit dialog or unsaved form is open.
    if (!userApproved || reloadHandled) return;
    reloadHandled = true;
    location.reload();
  });
  $('pwa-update-later').addEventListener('click', () => {
    dismissed = true;
    $('pwa-update-banner').hidden = true;
  });
  $('pwa-update-now').addEventListener('click', () => {
    if (!registration?.waiting) return;
    userApproved = true;
    $('pwa-update-now').disabled = true;
    $('pwa-update-now').textContent = 'Aktualisiere …';
    // Worker becomes active only after this explicit click.
    registration.waiting.postMessage({type:'SKIP_WAITING'});
  });
  window.addEventListener('load', async () => {
    try {
      registration = await navigator.serviceWorker.register('./sw.js', {updateViaCache:'none'});
      offer();
      watchInstalling(registration.installing);
      registration.addEventListener('updatefound', () => watchInstalling(registration.installing));
      await check();
      window.addEventListener('online', check);
      window.addEventListener('focus', check);
      document.addEventListener('visibilitychange', () => {if (!document.hidden) check();});
      window.setInterval(() => {if (!document.hidden) check();}, 30 * 60 * 1000);
    } catch (error) { console.warn('PWA-Registrierung:', error); }
  });
})();
