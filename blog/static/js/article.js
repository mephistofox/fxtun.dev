// Copy buttons on code blocks and the current section in the table of contents.
(function () {
  'use strict';
  document.addEventListener('click', function (e) {
    var btn = e.target.closest('.term button[data-copy]');
    if (!btn || !navigator.clipboard) return;
    var code = btn.closest('.term').querySelector('pre code');
    var text = code.innerText.replace(/^\$ /gm, '');
    navigator.clipboard.writeText(text).then(function () {
      btn.textContent = 'Скопировано';
      setTimeout(function () { btn.textContent = 'Копировать'; }, 1600);
    });
  });

  var links = document.querySelectorAll('.rail #TableOfContents a');
  if (!links.length || !('IntersectionObserver' in window)) return;
  var byId = {};
  links.forEach(function (a) { byId[decodeURIComponent(a.hash.slice(1))] = a; });
  var io = new IntersectionObserver(function (entries) {
    entries.forEach(function (en) {
      if (!en.isIntersecting) return;
      links.forEach(function (a) { a.classList.remove('on'); });
      var a = byId[en.target.id];
      if (a) a.classList.add('on');
    });
  }, { rootMargin: '-20% 0px -70% 0px' });
  Object.keys(byId).forEach(function (id) {
    var h = document.getElementById(id);
    if (h) io.observe(h);
  });
})();
