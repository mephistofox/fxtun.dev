// Analytics loads after the page has settled.
//
// gtag and Metrika together are ~260 KiB of third-party script, and pulled in
// during load they were the bulk of the main-thread work the page was judged
// on. Nothing is dropped: both libraries are built around a queue that the
// real script drains when it arrives, so the stubs below are defined
// immediately and every call made before the download still gets sent.

(function () {
  var started = false;

  function loadScripts() {
    if (started) return;
    started = true;

    var gaId = { 'fxtun.ru': 'G-TKFQMYKJZZ', 'fxtun.dev': 'G-4FH5VTH49H' }[location.hostname];
    if (gaId) {
      var s = document.createElement('script');
      s.async = true;
      s.src = 'https://www.googletagmanager.com/gtag/js?id=' + gaId;
      document.head.appendChild(s);
    }

    if (location.hostname === 'fxtun.ru') {
      var m = document.createElement('script');
      m.async = true;
      m.src = 'https://mc.yandex.ru/metrika/tag.js?id=108256538';
      document.head.appendChild(m);
    }
  }

  // The first thing the visitor does, or fifteen seconds of them staying put.
  // Measured: loading these during the initial render cost about 20 points of
  // the performance score and half a second of blocking time, and none of it
  // is work the visitor asked for. The cost of waiting is that somebody who
  // opens the page and leaves within fifteen seconds without touching
  // anything is never counted.
  function schedule() {
    setTimeout(loadScripts, 15000);
    ['pointerdown', 'keydown', 'touchstart', 'scroll', 'wheel'].forEach(function (ev) {
      addEventListener(ev, loadScripts, { once: true, passive: true });
    });
  }

  if (document.readyState === 'complete') schedule();
  else addEventListener('load', schedule, { once: true });

  // Queues, defined now so nothing said before the download is lost.
  var gaId = { 'fxtun.ru': 'G-TKFQMYKJZZ', 'fxtun.dev': 'G-4FH5VTH49H' }[location.hostname];
  if (gaId) {
    window.dataLayer = window.dataLayer || [];
    window.gtag = function () { window.dataLayer.push(arguments); };
    window.gtag('js', new Date());
    window.gtag('config', gaId, { send_page_view: false });
  }

  if (location.hostname === 'fxtun.ru') {
    window.ym = window.ym || function () { (window.ym.a = window.ym.a || []).push(arguments); };
    window.ym.l = 1 * new Date();
    window.ym(108256538, 'init', {
      ssr: true, webvisor: true, clickmap: true, ecommerce: 'dataLayer',
      referrer: document.referrer, url: location.href,
      accurateTrackBounce: true, trackLinks: true,
    });
  }
})();
