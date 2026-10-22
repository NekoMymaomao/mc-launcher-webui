const API_BASE = (window.location.protocol.startsWith('http') && window.location.port !== '9876')
    ? `${window.location.protocol}//${window.location.hostname}:9876`
    : '';

async function api(endpoint, options = {}) {
    const url = `${API_BASE}/api${endpoint}`;
    const defaultOptions = {
        headers: { 'Content-Type': 'application/json' },
        ...options
    };
    if (defaultOptions.body && typeof defaultOptions.body === 'object') {
        defaultOptions.body = JSON.stringify(defaultOptions.body);
    }
    try {
        const resp = await fetch(url, defaultOptions);
        return await resp.json();
    } catch (e) {
        return { ok: false, error: `请求失败: ${e.message}` };
    }
}

const API = {
    ping: () => api('/ping'),
    info: () => api('/info'),

    versions: {
        list: () => api('/versions'),
        install: (v) => api('/versions/install', { method: 'POST', body: { version: v } }),
        remove: (v) => api('/versions/remove', { method: 'POST', body: { version: v } }),
        verify: (v) => api('/versions/verify', { method: 'POST', body: { version: v } })
    },

    launch: (v, u) => api('/launch', { method: 'POST', body: { version: v, username: u } }),

    accounts: {
        list: () => api('/accounts'),
        offline: (n) => api('/accounts/offline', { method: 'POST', body: { name: n } }),
        select: (uuid) => api('/accounts', { method: 'PUT', body: { uuid } }),
        remove: (uuid) => api(`/accounts?uuid=${encodeURIComponent(uuid)}`, { method: 'DELETE' }),
        microsoftStart: () => api('/accounts/microsoft/start', { method: 'POST' }),
        microsoftPoll: (deviceCode) => api('/accounts/microsoft/poll', { method: 'POST', body: { deviceCode } })
    },

    java: {
        list: () => api('/java'),
        install: (m) => api('/java/install', { method: 'POST', body: { major: m } }),
        set: (p) => api('/java/set', { method: 'PUT', body: { path: p } })
    },

    modrinth: {
        search: (q, t, p) => api(`/modrinth/search?q=${encodeURIComponent(q)}&type=${t || ''}&page=${p || 1}`),
        info: (s) => api(`/modrinth/info/${encodeURIComponent(s)}`),
        versions: (s, gv, ld) => api(`/modrinth/versions?slug=${encodeURIComponent(s)}&game_version=${gv || ''}&loader=${ld || ''}`),
        install: (data) => api('/modrinth/install', { method: 'POST', body: data })
    },

    config: {
        get: () => api('/config'),
        set: (c) => api('/config', { method: 'PUT', body: c })
    },

    servers: {
        list: () => api('/servers'),
        add: (a, n) => api('/servers', { method: 'POST', body: { address: a, name: n } }),
        remove: (a) => api(`/servers/${encodeURIComponent(a)}`, { method: 'DELETE' })
    },

    docs: {
        api: () => fetch(`${API_BASE}/docs/api.html`).then(r => r.text()),
        tutorial: () => fetch(`${API_BASE}/docs/tutorial.html`).then(r => r.text()),
        faq: () => fetch(`${API_BASE}/docs/faq.html`).then(r => r.text())
    }
};
