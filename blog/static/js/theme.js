// Theme management — compatible with main fxTunnel site
// Uses same localStorage key 'theme' for seamless transition

(function() {
  'use strict';

  var STORAGE_KEY = 'theme';
  var modes = ['light', 'dark', 'system'];

  function getStoredTheme() {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch (e) {
      return null;
    }
  }

  function setStoredTheme(mode) {
    try {
      localStorage.setItem(STORAGE_KEY, mode);
    } catch (e) {}
  }

  function getSystemPreference() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function resolveTheme(mode) {
    if (mode === 'dark') return 'dark';
    if (mode === 'light') return 'light';
    return getSystemPreference();
  }

  function applyTheme(mode) {
    var resolved = resolveTheme(mode);
    document.documentElement.classList.toggle('dark', resolved === 'dark');
    updateIcons(mode);
  }

  function updateIcons(mode) {
    var sunIcon = document.getElementById('theme-icon-sun');
    var moonIcon = document.getElementById('theme-icon-moon');
    var monitorIcon = document.getElementById('theme-icon-monitor');
    if (!sunIcon) return;

    sunIcon.style.display = mode === 'light' ? 'block' : 'none';
    moonIcon.style.display = mode === 'dark' ? 'block' : 'none';
    monitorIcon.style.display = (mode === 'system' || !mode) ? 'block' : 'none';
  }

  function getCurrentMode() {
    var stored = getStoredTheme();
    if (stored && modes.indexOf(stored) !== -1) return stored;
    return 'system';
  }

  window.cycleTheme = function() {
    var current = getCurrentMode();
    var idx = modes.indexOf(current);
    var next = modes[(idx + 1) % modes.length];
    setStoredTheme(next);
    applyTheme(next);
  };

  // Listen for system preference changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
    if (getCurrentMode() === 'system') {
      applyTheme('system');
    }
  });

  // Apply on DOM ready (icons update)
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      applyTheme(getCurrentMode());
    });
  } else {
    applyTheme(getCurrentMode());
  }
})();
