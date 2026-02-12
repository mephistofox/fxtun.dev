/* ===========================================================
   fxTunnel Inspector — Vanilla JS Application
   =========================================================== */

// --------------- State ---------------
var exchanges = [];
var selectedId = null;
var selectedDetail = null;
var filters = { method: '', status: '', path: '' };
var locale = detectLocale();
var darkMode = detectTheme();
var eventSource = null;
var summaryTimer = null;

// --------------- i18n ---------------
var i18n = {
    en: {
        title: 'Inspector',
        requests: 'Requests',
        noRequests: 'No requests captured yet',
        selectRequest: 'Select a request to view details',
        request: 'Request',
        response: 'Response',
        headers: 'Headers',
        body: 'Body',
        replay: 'Replay',
        clear: 'Clear All',
        filter: 'Filter...',
        allMethods: 'All Methods',
        allStatuses: 'All Statuses',
        total: 'Total',
        errors: 'Errors',
        avgTime: 'Avg Time',
        status: 'Status',
        tunnels: 'Tunnels',
        noTunnels: 'No active tunnels',
        requestHeaders: 'Request Headers',
        responseHeaders: 'Response Headers',
        requestBody: 'Request Body',
        responseBody: 'Response Body',
        noBody: '(empty body)',
        general: 'General',
        replaying: 'Replaying...',
        replayOk: 'Replayed successfully',
        replayFail: 'Replay failed',
    },
    ru: {
        title: 'Инспектор',
        requests: 'Запросы',
        noRequests: 'Запросы пока не перехвачены',
        selectRequest: 'Выберите запрос для просмотра',
        request: 'Запрос',
        response: 'Ответ',
        headers: 'Заголовки',
        body: 'Тело',
        replay: 'Повторить',
        clear: 'Очистить все',
        filter: 'Фильтр...',
        allMethods: 'Все методы',
        allStatuses: 'Все статусы',
        total: 'Всего',
        errors: 'Ошибки',
        avgTime: 'Среднее',
        status: 'Статус',
        tunnels: 'Туннели',
        noTunnels: 'Нет активных туннелей',
        requestHeaders: 'Заголовки запроса',
        responseHeaders: 'Заголовки ответа',
        requestBody: 'Тело запроса',
        responseBody: 'Тело ответа',
        noBody: '(пустое тело)',
        general: 'Основное',
        replaying: 'Повтор...',
        replayOk: 'Успешно повторено',
        replayFail: 'Ошибка повтора',
    }
};

function t(key) { return (i18n[locale] && i18n[locale][key]) || key; }

// --------------- Theme ---------------
function detectTheme() {
    var saved = localStorage.getItem('fxtunnel-theme');
    if (saved) return saved === 'dark';
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function applyTheme() {
    document.body.classList.toggle('dark', darkMode);
    var sun = document.getElementById('icon-sun');
    var moon = document.getElementById('icon-moon');
    if (sun && moon) {
        sun.classList.toggle('hidden', darkMode);
        moon.classList.toggle('hidden', !darkMode);
    }
}

function toggleTheme() {
    darkMode = !darkMode;
    localStorage.setItem('fxtunnel-theme', darkMode ? 'dark' : 'light');
    applyTheme();
}

// --------------- Locale ---------------
function detectLocale() {
    var saved = localStorage.getItem('fxtunnel-locale');
    if (saved) return saved;
    return navigator.language.startsWith('ru') ? 'ru' : 'en';
}

function applyLocale() {
    var label = document.getElementById('locale-label');
    if (label) label.textContent = locale.toUpperCase();

    document.querySelectorAll('[data-i18n]').forEach(function(el) {
        el.textContent = t(el.getAttribute('data-i18n'));
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(function(el) {
        el.placeholder = t(el.getAttribute('data-i18n-placeholder'));
    });
}

function toggleLocale() {
    locale = locale === 'en' ? 'ru' : 'en';
    localStorage.setItem('fxtunnel-locale', locale);
    applyLocale();
    // Re-render detail if selected
    if (selectedDetail) {
        renderExchangeDetail(selectedDetail);
    }
}

// --------------- API ---------------
function fetchExchanges() {
    return fetch('/api/requests/http?limit=100')
        .then(function(r) { return r.json(); })
        .then(function(data) {
            exchanges = data.requests || [];
            renderExchangeList();
        })
        .catch(function() {});
}

function fetchExchange(id) {
    return fetch('/api/requests/http/' + id)
        .then(function(r) {
            if (!r.ok) throw new Error('Not found');
            return r.json();
        })
        .then(function(data) {
            selectedDetail = data;
            renderExchangeDetail(data);
        })
        .catch(function() {});
}

function fetchSummary() {
    return fetch('/api/requests/http/summary')
        .then(function(r) { return r.json(); })
        .then(function(data) { renderSummary(data); })
        .catch(function() {});
}

function deleteExchanges() {
    return fetch('/api/requests/http', { method: 'DELETE' })
        .then(function() {
            exchanges = [];
            selectedId = null;
            selectedDetail = null;
            renderExchangeList();
            renderEmptyDetail();
            fetchSummary();
        })
        .catch(function() {});
}

function replayExchange(id) {
    var btn = document.getElementById('btn-replay');
    if (btn) {
        btn.textContent = t('replaying');
        btn.disabled = true;
    }

    return fetch('/api/requests/http', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: id })
    })
    .then(function(r) {
        if (!r.ok) throw new Error('Replay failed');
        return r.json();
    })
    .then(function(data) {
        if (btn) btn.textContent = t('replayOk');
        // New exchange will arrive via SSE; optionally select it
        if (data.exchange_id) {
            setTimeout(function() { selectExchange(data.exchange_id); }, 300);
        }
    })
    .catch(function() {
        if (btn) btn.textContent = t('replayFail');
    })
    .finally(function() {
        setTimeout(function() {
            if (btn) {
                btn.textContent = t('replay');
                btn.disabled = false;
            }
        }, 1500);
    });
}

// --------------- SSE ---------------
function connectSSE() {
    if (eventSource) {
        eventSource.close();
    }

    eventSource = new EventSource('/api/requests/http/stream');

    eventSource.addEventListener('exchange', function(e) {
        try {
            var ex = JSON.parse(e.data);
            // SSE sends summary objects (no bodies), merge into list
            // Prepend to array (newest first)
            exchanges.unshift(ex);
            // Cap at 500 entries in the UI list
            if (exchanges.length > 500) exchanges.length = 500;
            renderExchangeList();
            fetchSummary();
        } catch (_) {}
    });

    eventSource.onerror = function() {
        // Reconnect after a delay
        eventSource.close();
        setTimeout(connectSSE, 3000);
    };
}

// --------------- Rendering: List ---------------
function getFilteredExchanges() {
    return exchanges.filter(function(ex) {
        if (filters.method && ex.method !== filters.method) return false;
        if (filters.status) {
            if (!matchStatusFilter(ex.status_code, filters.status)) return false;
        }
        if (filters.path) {
            var search = filters.path.toLowerCase();
            var path = (ex.path || '').toLowerCase();
            if (path.indexOf(search) === -1) return false;
        }
        return true;
    });
}

function matchStatusFilter(code, filter) {
    switch (filter) {
        case '2xx': return code >= 200 && code < 300;
        case '3xx': return code >= 300 && code < 400;
        case '4xx': return code >= 400 && code < 500;
        case '5xx': return code >= 500 && code < 600;
        default: return true;
    }
}

function renderExchangeList() {
    var container = document.getElementById('exchange-list');
    var empty = document.getElementById('empty-state');

    var filtered = getFilteredExchanges();

    if (filtered.length === 0) {
        // Remove all rows but keep empty state
        container.querySelectorAll('.exchange-row').forEach(function(r) { r.remove(); });
        if (empty) empty.classList.remove('hidden');
        return;
    }

    if (empty) empty.classList.add('hidden');

    // Build HTML
    var html = '';
    filtered.forEach(function(ex) {
        var id = ex.id;
        var method = (ex.method || 'GET').toUpperCase();
        var path = ex.path || '/';
        var status = ex.status_code || 0;
        var duration = ex.duration_ms !== undefined ? ex.duration_ms : Math.round((ex.duration_ns || 0) / 1e6);
        var ts = ex.timestamp ? formatTime(ex.timestamp) : '';

        var methodClass = 'method-' + method.toLowerCase();
        var statusClass = statusCssClass(status);
        var activeClass = id === selectedId ? ' active' : '';

        html += '<div class="exchange-row' + activeClass + '" data-id="' + escapeAttr(id) + '">'
            + '<span class="method-badge ' + methodClass + '">' + escapeHtml(method) + '</span>'
            + '<span class="exchange-path" title="' + escapeAttr(path) + '">' + escapeHtml(path) + '</span>'
            + '<div class="exchange-meta">'
            + '<span class="status-badge ' + statusClass + '">' + (status || '-') + '</span>'
            + '<span class="exchange-duration">' + duration + 'ms</span>'
            + '<span class="exchange-time">' + escapeHtml(ts) + '</span>'
            + '</div>'
            + '</div>';
    });

    container.innerHTML = html;

    // Keep empty state element
    var emptyDiv = document.createElement('div');
    emptyDiv.className = 'empty-state hidden';
    emptyDiv.id = 'empty-state';
    emptyDiv.innerHTML = '<svg viewBox="0 0 24 24" width="48" height="48" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">'
        + '<path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>'
        + '<p data-i18n="noRequests">' + t('noRequests') + '</p>';
    container.appendChild(emptyDiv);

    // Click handlers
    container.querySelectorAll('.exchange-row').forEach(function(row) {
        row.addEventListener('click', function() {
            selectExchange(this.getAttribute('data-id'));
        });
    });
}

function selectExchange(id) {
    selectedId = id;
    // Highlight in list
    document.querySelectorAll('.exchange-row').forEach(function(row) {
        row.classList.toggle('active', row.getAttribute('data-id') === id);
    });
    fetchExchange(id);
}

// --------------- Rendering: Detail ---------------
function renderEmptyDetail() {
    document.getElementById('detail-empty').classList.remove('hidden');
    document.getElementById('detail-content').classList.add('hidden');
}

function renderExchangeDetail(ex) {
    document.getElementById('detail-empty').classList.add('hidden');
    document.getElementById('detail-content').classList.remove('hidden');

    var method = (ex.method || 'GET').toUpperCase();
    var methodClass = 'method-' + method.toLowerCase();
    var status = ex.status_code || 0;
    var statusClass = statusCssClass(status);
    var duration = ex.duration_ms !== undefined ? ex.duration_ms : Math.round((ex.duration_ns || 0) / 1e6);

    // Header
    var headerEl = document.getElementById('detail-header');
    headerEl.innerHTML =
        '<span class="method-badge ' + methodClass + '">' + escapeHtml(method) + '</span>'
        + '<span class="detail-url">' + escapeHtml(ex.host || '') + escapeHtml(ex.path || '/') + '</span>'
        + '<span class="detail-status status-badge ' + statusClass + '">' + (status || '-') + '</span>'
        + '<span class="detail-duration">' + duration + 'ms</span>';

    // Request tab
    var reqTab = document.getElementById('tab-request');
    var reqBodyHtml = renderBody(ex.request_body, ex.request_body_size);
    reqTab.innerHTML =
        '<div class="headers-section"><h3>' + t('requestHeaders') + '</h3>'
        + renderHeadersTable(ex.request_headers)
        + '</div>'
        + '<div class="body-section"><h3>' + t('requestBody') + '</h3>'
        + reqBodyHtml
        + '</div>';

    // Response tab
    var respTab = document.getElementById('tab-response');
    var respBodyHtml = renderBody(ex.response_body, ex.response_body_size);
    respTab.innerHTML =
        '<div class="headers-section"><h3>' + t('responseHeaders') + '</h3>'
        + renderHeadersTable(ex.response_headers)
        + '</div>'
        + '<div class="body-section"><h3>' + t('responseBody') + '</h3>'
        + respBodyHtml
        + '</div>';

    // Headers tab (combined)
    var hdrsTab = document.getElementById('tab-headers');
    hdrsTab.innerHTML =
        '<div class="headers-section"><h3>' + t('general') + '</h3>'
        + '<div class="info-row"><span class="info-label">URL</span><span class="info-value">' + escapeHtml((ex.host || '') + (ex.path || '/')) + '</span></div>'
        + '<div class="info-row"><span class="info-label">Method</span><span class="info-value">' + escapeHtml(method) + '</span></div>'
        + '<div class="info-row"><span class="info-label">Status</span><span class="info-value">' + (status || '-') + '</span></div>'
        + '<div class="info-row"><span class="info-label">Duration</span><span class="info-value">' + duration + 'ms</span></div>'
        + (ex.remote_addr ? '<div class="info-row"><span class="info-label">Remote</span><span class="info-value">' + escapeHtml(ex.remote_addr) + '</span></div>' : '')
        + (ex.tunnel_id ? '<div class="info-row"><span class="info-label">Tunnel</span><span class="info-value">' + escapeHtml(ex.tunnel_id) + '</span></div>' : '')
        + '</div>'
        + '<div class="headers-section"><h3>' + t('requestHeaders') + '</h3>'
        + renderHeadersTable(ex.request_headers)
        + '</div>'
        + '<div class="headers-section"><h3>' + t('responseHeaders') + '</h3>'
        + renderHeadersTable(ex.response_headers)
        + '</div>';

    // Replay button
    var replayBtn = document.getElementById('btn-replay');
    replayBtn.textContent = t('replay');
    replayBtn.onclick = function() { replayExchange(ex.id); };
}

function renderHeadersTable(headers) {
    if (!headers || Object.keys(headers).length === 0) {
        return '<span class="body-empty">-</span>';
    }

    var html = '<table class="headers-table">';
    var keys = Object.keys(headers).sort();
    keys.forEach(function(key) {
        var vals = headers[key];
        // headers can be {string: string[]} (Go http.Header) or {string: string}
        var value = Array.isArray(vals) ? vals.join(', ') : String(vals);
        html += '<tr><td>' + escapeHtml(key) + '</td><td>' + escapeHtml(value) + '</td></tr>';
    });
    html += '</table>';
    return html;
}

function renderBody(body, size) {
    if (!body || (typeof size === 'number' && size === 0)) {
        return '<div class="body-content body-empty">' + t('noBody') + '</div>';
    }

    var text = '';
    // The detail API returns raw bytes as base64-encoded JSON byte array or string
    if (typeof body === 'string') {
        // Could be base64 encoded
        try {
            text = decodeBase64(body);
        } catch (_) {
            text = body;
        }
    } else if (Array.isArray(body)) {
        // JSON byte array [number, ...]
        try {
            text = String.fromCharCode.apply(null, body);
        } catch (_) {
            text = String(body);
        }
    } else {
        text = String(body);
    }

    // Attempt to pretty-print JSON
    var highlighted = tryFormatJSON(text);
    if (highlighted !== null) {
        return '<div class="body-content">' + highlighted + '</div>';
    }

    return '<div class="body-content">' + escapeHtml(text) + '</div>';
}

function decodeBase64(str) {
    // Handle both standard and URL-safe base64
    try {
        return atob(str);
    } catch (_) {
        // Try URL-safe
        var s = str.replace(/-/g, '+').replace(/_/g, '/');
        while (s.length % 4) s += '=';
        return atob(s);
    }
}

// --------------- JSON Highlighting ---------------
function tryFormatJSON(text) {
    var trimmed = text.trim();
    if ((trimmed[0] !== '{' && trimmed[0] !== '[') || trimmed.length === 0) return null;

    try {
        var obj = JSON.parse(trimmed);
        var pretty = JSON.stringify(obj, null, 2);
        return highlightJSON(pretty);
    } catch (_) {
        return null;
    }
}

function highlightJSON(str) {
    // Regex-based JSON syntax highlighting
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"([^"\\]*(\\.[^"\\]*)*)"\s*:/g, '<span class="json-key">"$1"</span>:')
        .replace(/:\s*"([^"\\]*(\\.[^"\\]*)*)"/g, ': <span class="json-string">"$1"</span>')
        .replace(/:\s*(-?\d+\.?\d*([eE][+-]?\d+)?)/g, ': <span class="json-number">$1</span>')
        .replace(/:\s*(true|false)/g, ': <span class="json-bool">$1</span>')
        .replace(/:\s*(null)/g, ': <span class="json-null">$1</span>');
}

// --------------- Rendering: Summary ---------------
function renderSummary(data) {
    var totalEl = document.getElementById('stat-total');
    var errorsEl = document.getElementById('stat-errors');
    var avgEl = document.getElementById('stat-avg');

    if (totalEl) totalEl.textContent = data.total || 0;
    if (errorsEl) {
        var rate = data.error_rate || 0;
        errorsEl.textContent = (rate * 100).toFixed(1) + '%';
    }
    if (avgEl) avgEl.textContent = (data.avg_duration_ms || 0) + 'ms';
}

// --------------- Helpers ---------------
function statusCssClass(code) {
    if (code >= 200 && code < 300) return 'status-2xx';
    if (code >= 300 && code < 400) return 'status-3xx';
    if (code >= 400 && code < 500) return 'status-4xx';
    if (code >= 500 && code < 600) return 'status-5xx';
    return 'status-0';
}

function formatTime(ts) {
    try {
        var d = new Date(ts);
        var h = String(d.getHours()).padStart(2, '0');
        var m = String(d.getMinutes()).padStart(2, '0');
        var s = String(d.getSeconds()).padStart(2, '0');
        return h + ':' + m + ':' + s;
    } catch (_) {
        return '';
    }
}

function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function escapeAttr(str) {
    return escapeHtml(str);
}

// --------------- Tabs ---------------
function initTabs() {
    document.querySelectorAll('.detail-tabs .tab').forEach(function(tab) {
        tab.addEventListener('click', function() {
            var target = this.getAttribute('data-tab');

            document.querySelectorAll('.detail-tabs .tab').forEach(function(t) {
                t.classList.remove('active');
            });
            this.classList.add('active');

            document.querySelectorAll('.tab-pane').forEach(function(p) {
                p.classList.remove('active');
            });
            document.getElementById('tab-' + target).classList.add('active');
        });
    });
}

// --------------- Filters ---------------
function initFilters() {
    var methodSelect = document.getElementById('filter-method');
    var statusSelect = document.getElementById('filter-status');
    var pathInput = document.getElementById('filter-path');
    var clearBtn = document.getElementById('btn-clear');

    methodSelect.addEventListener('change', function() {
        filters.method = this.value;
        renderExchangeList();
    });

    statusSelect.addEventListener('change', function() {
        filters.status = this.value;
        renderExchangeList();
    });

    var pathDebounce = null;
    pathInput.addEventListener('input', function() {
        var val = this.value;
        clearTimeout(pathDebounce);
        pathDebounce = setTimeout(function() {
            filters.path = val;
            renderExchangeList();
        }, 200);
    });

    clearBtn.addEventListener('click', function() {
        deleteExchanges();
    });
}

// --------------- Init ---------------
function init() {
    applyTheme();
    applyLocale();
    initTabs();
    initFilters();

    document.getElementById('theme-toggle').addEventListener('click', toggleTheme);
    document.getElementById('locale-toggle').addEventListener('click', toggleLocale);

    // Load initial data
    fetchExchanges();
    fetchSummary();

    // Connect SSE for live updates
    connectSSE();

    // Periodic summary refresh
    summaryTimer = setInterval(fetchSummary, 5000);
}

document.addEventListener('DOMContentLoaded', init);
