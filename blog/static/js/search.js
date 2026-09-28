(function() {
  'use strict';

  var searchInput = document.getElementById('search-input');
  var searchResults = document.getElementById('search-results');
  if (!searchInput || !searchResults) return;

  var searchIndex = null;
  var debounceTimer = null;

  // Determine base URL from the page
  var baseEl = document.querySelector('link[rel="canonical"]');
  var baseURL = '';
  if (baseEl) {
    var href = baseEl.getAttribute('href');
    // Extract base: find /blog/ or /blog/en/
    var match = href.match(/(.*\/blog\/(?:en\/)?)/);
    if (match) baseURL = match[1];
  }

  // Determine language from html lang attribute
  var lang = document.documentElement.lang || 'ru';
  var indexURL = baseURL || window.location.origin + '/blog/';
  // For EN, the index is at /blog/en/index.json
  if (lang === 'en') {
    indexURL = indexURL.replace(/\/$/, '') + '/index.json';
  } else {
    // For RU (default), index is at /blog/index.json
    indexURL = indexURL.replace(/\/?(en\/?)?$/, '/index.json');
  }

  function loadIndex(cb) {
    if (searchIndex) return cb(searchIndex);
    var xhr = new XMLHttpRequest();
    xhr.open('GET', indexURL, true);
    xhr.onreadystatechange = function() {
      if (xhr.readyState === 4 && xhr.status === 200) {
        try {
          searchIndex = JSON.parse(xhr.responseText);
        } catch(e) {
          searchIndex = [];
        }
        cb(searchIndex);
      }
    };
    xhr.send();
  }

  function search(query) {
    if (!query || query.length < 2) {
      searchResults.innerHTML = '';
      searchResults.style.display = 'none';
      return;
    }

    loadIndex(function(index) {
      var q = query.toLowerCase();
      var results = index.filter(function(item) {
        return (
          item.title.toLowerCase().indexOf(q) !== -1 ||
          item.description.toLowerCase().indexOf(q) !== -1 ||
          item.content.toLowerCase().indexOf(q) !== -1 ||
          (item.tags && item.tags.some(function(t) { return t.toLowerCase().indexOf(q) !== -1; }))
        );
      });

      if (results.length === 0) {
        searchResults.innerHTML = '<div class="search-empty">' +
          (lang === 'ru' ? 'Ничего не найдено' : 'No results found') +
          '</div>';
        searchResults.style.display = 'block';
        return;
      }

      var html = results.slice(0, 5).map(function(item) {
        return '<a href="' + item.url + '" class="search-result-item">' +
          '<span class="search-result-title">' + escapeHtml(item.title) + '</span>' +
          '<span class="search-result-desc">' + escapeHtml(item.description).substring(0, 80) + '</span>' +
          '</a>';
      }).join('');

      searchResults.innerHTML = html;
      searchResults.style.display = 'block';
    });
  }

  function escapeHtml(str) {
    var div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  searchInput.addEventListener('input', function() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function() {
      search(searchInput.value.trim());
    }, 200);
  });

  // Close results on click outside
  document.addEventListener('click', function(e) {
    if (!searchInput.contains(e.target) && !searchResults.contains(e.target)) {
      searchResults.style.display = 'none';
    }
  });

  // Reopen on focus if there's a query
  searchInput.addEventListener('focus', function() {
    if (searchInput.value.trim().length >= 2) {
      search(searchInput.value.trim());
    }
  });
})();
