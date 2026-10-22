const app = document.getElementById('app');
let currentTab = 'launch';

document.querySelectorAll('.tab').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        currentTab = btn.dataset.tab;
        render();
    });
});

async function render() {
    switch (currentTab) {
        case 'launch': await renderLaunch(); break;
        case 'versions': await renderVersions(); break;
        case 'mods': renderMods(); break;
        case 'accounts': await renderAccounts(); break;
        case 'config': await renderConfig(); break;
        case 'docs': await renderDocs(); break;
    }
}

function errText(r) {
    if (!r) return '未知错误';
    return r.error || (r.data && r.data.message) || '未知错误';
}

// ---------- 启动 ----------

async function renderLaunch() {
    const versions = await API.versions.list();
    const data = versions.data || [];
    app.innerHTML = `
        <h2>${t('launch')}</h2>
        ${data.length === 0 ? `<p>${t('noVersion')}</p>` : `
        <div class="form-group">
            <label>${t('version')}</label>
            <select id="versionSelect">${data.map(v => `<option value="${v}">${v}</option>`).join('')}</select>
        </div>
        <div class="form-group">
            <label>${t('username')}</label>
            <input type="text" id="username" value="Player">
        </div>
        <button class="btn" onclick="doLaunch()" style="width:100%;padding:20px;font-size:20px">${t('start')}</button>
        <div id="launchResult" style="margin-top:12px"></div>`}
    `;
}

async function doLaunch() {
    const sel = document.getElementById('versionSelect');
    if (!sel) return;
    const v = sel.value;
    const u = document.getElementById('username').value || 'Player';
    const box = document.getElementById('launchResult');
    box.textContent = '正在启动...';
    const r = await API.launch(v, u);
    box.textContent = r.ok ? '已启动，请查看服务端控制台输出' : `启动失败: ${errText(r)}`;
}

// ---------- 版本 ----------

async function renderVersions() {
    const versions = await API.versions.list();
    const data = versions.data || [];
    app.innerHTML = `
        <h2>${t('versions')}</h2>
        ${data.length === 0 ? `<p>${t('noVersion')}</p>` :
            data.map(v => `<div class="list-item">
                <span>${v}</span>
                <div>
                    <button onclick="doVerify('${v}')">${t('verify')}</button>
                    <button class="btn-danger" onclick="doRemove('${v}')">${t('remove')}</button>
                </div>
            </div>`).join('')}
        <div class="modal" id="installModal">
            <div class="modal-content">
                <span class="close" onclick="closeModal()">&times;</span>
                <h3>${t('install')}</h3>
                <input type="text" id="newVersion" placeholder="1.20.1">
                <button class="btn" onclick="doInstall()" style="width:100%;margin-top:10px">${t('install')}</button>
                <div id="installResult" style="margin-top:10px"></div>
            </div>
        </div>
        <button class="btn" onclick="document.getElementById('installModal').classList.add('show')" style="margin-top:20px;width:100%">${t('install')}</button>
    `;
}

async function doInstall() {
    const v = document.getElementById('newVersion').value.trim();
    if (!v) return;
    const box = document.getElementById('installResult');
    box.textContent = '正在安装，请稍候（首次安装可能较久）...';
    const r = await API.versions.install(v);
    box.textContent = r.ok ? '安装完成' : `安装失败: ${errText(r)}`;
    if (r.ok) setTimeout(() => { closeModal(); renderVersions(); }, 800);
}

async function doRemove(v) {
    if (!confirm(`删除 ${v}?`)) return;
    const r = await API.versions.remove(v);
    if (!r.ok) alert(`删除失败: ${errText(r)}`);
    renderVersions();
}

async function doVerify(v) {
    const r = await API.versions.verify(v);
    if (r.ok) {
        alert((r.data && r.data.message) || '验证通过');
    } else {
        alert(`验证失败: ${errText(r)}`);
    }
}

function closeModal() {
    document.querySelectorAll('.modal').forEach(m => m.classList.remove('show'));
}

// ---------- 模组 ----------

function renderMods() {
    app.innerHTML = `
        <h2>${t('mods')}</h2>
        <div class="form-group">
            <input type="text" id="searchQuery" placeholder="${t('search')}...">
            <button class="btn" onclick="searchMods()" style="margin-top:10px">${t('search')}</button>
        </div>
        <div id="modResults"></div>
    `;
}

async function searchMods() {
    const q = document.getElementById('searchQuery').value.trim();
    if (!q) return;
    const box = document.getElementById('modResults');
    box.textContent = '搜索中...';
    const r = await API.modrinth.search(q, 'mod');
    if (!r.ok) { box.textContent = `搜索失败: ${errText(r)}`; return; }
    const results = (r.data && r.data.hits) || [];
    if (results.length === 0) { box.textContent = '没有结果'; return; }
    box.innerHTML = results.map(m => `
        <div class="list-item">
            <div>
                <strong>${m.title}</strong><br>
                <small>${(m.description || '').substring(0, 80)}</small>
            </div>
            <button class="btn" onclick="installMod('${m.slug}')">${t('install')}</button>
        </div>
    `).join('');
}

async function installMod(slug) {
    const r = await API.modrinth.install({ slug, type: 'mods' });
    if (r.ok) {
        alert(`已下载到: ${r.data.file}`);
    } else {
        alert(`安装失败: ${errText(r)}`);
    }
}

// ---------- 账号 ----------

async function renderAccounts() {
    const r = await API.accounts.list();
    const list = (r.data && r.data.accounts) || [];
    const cur = r.data && r.data.current;
    app.innerHTML = `
        <h2>${t('accounts')}</h2>
        ${list.length === 0 ? '<p>暂无账号</p>' : list.map(a => `
            <div class="list-item">
                <span>${a.name} (${a.type})${cur && cur.uuid === a.uuid ? ' ✓' : ''}</span>
                <div>
                    ${cur && cur.uuid === a.uuid ? '' : `<button onclick="useAccount('${a.uuid}')">使用</button>`}
                    <button class="btn-danger" onclick="delAccount('${a.uuid}')">删除</button>
                </div>
            </div>`).join('')}
        <button class="btn" onclick="doOffline()" style="margin-top:10px">${t('offlineLogin')}</button>
        <button class="btn" onclick="doMS()" style="margin-top:10px;margin-left:10px">${t('msLogin')}</button>
        <div id="msBox" style="margin-top:12px"></div>
    `;
}

async function doOffline() {
    const name = prompt('输入用户名:');
    if (name) {
        const r = await API.accounts.offline(name);
        if (!r.ok) alert(`登录失败: ${errText(r)}`);
        renderAccounts();
    }
}

async function useAccount(uuid) {
    const r = await API.accounts.select(uuid);
    if (!r.ok) alert(`切换失败: ${errText(r)}`);
    renderAccounts();
}

async function delAccount(uuid) {
    if (!confirm('删除该账号?')) return;
    await API.accounts.remove(uuid);
    renderAccounts();
}

async function doMS() {
    const box = document.getElementById('msBox');
    box.innerHTML = '正在申请设备码...';
    const start = await API.accounts.microsoftStart();
    if (!start.ok) {
        box.innerHTML = `申请设备码失败: ${errText(start)}`;
        return;
    }
    const dc = start.data;
    box.innerHTML = `
        <div class="list-item" style="display:block">
            <div>请打开 <a href="${dc.verificationUri}" target="_blank">${dc.verificationUri}</a></div>
            <div style="font-size:28px;letter-spacing:6px;margin:8px 0">${dc.userCode}</div>
            <div>输入上面的代码并完成登录，此页面会自动继续。</div>
            <div id="msStatus" style="margin-top:8px">等待授权...</div>
        </div>`;

    const deadline = Date.now() + (dc.expiresIn || 900) * 1000;
    const interval = Math.max(dc.interval || 5, 5) * 1000;
    while (Date.now() < deadline) {
        await new Promise(r => setTimeout(r, interval));
        const r = await API.accounts.microsoftPoll(dc.deviceCode);
        if (!r.ok) {
            document.getElementById('msStatus').textContent = `登录失败: ${errText(r)}`;
            return;
        }
        if (r.data && r.data.pending) continue;
        document.getElementById('msStatus').textContent = `登录成功: ${r.data.account.name}`;
        setTimeout(renderAccounts, 800);
        return;
    }
    document.getElementById('msStatus').textContent = '设备码已过期，请重新点击微软登录';
}

// ---------- 配置 ----------

async function renderConfig() {
    const cfg = await API.config.get();
    const c = cfg.data || {};
    app.innerHTML = `
        <h2>${t('config')}</h2>
        <div class="form-group"><label>最大内存 (MB)</label><input type="number" id="maxMem" value="${c.maxMemory || 2048}"></div>
        <div class="form-group"><label>最小内存 (MB)</label><input type="number" id="minMem" value="${c.minMemory || 1024}"></div>
        <div class="form-group"><label>下载线程</label><input type="number" id="threads" value="${c.threads || 8}"></div>
        <div class="form-group"><label>宽度</label><input type="number" id="width" value="${c.width || 854}"></div>
        <div class="form-group"><label>高度</label><input type="number" id="height" value="${c.height || 480}"></div>
        <div class="form-group"><label>Java 路径</label><input type="text" id="javaPath" value="${c.javaPath || ''}"></div>
        <div class="form-group"><label>语言</label>
            <select id="langSel">
                <option value="zh" ${c.lang === 'zh' ? 'selected' : ''}>中文</option>
                <option value="en" ${c.lang !== 'zh' ? 'selected' : ''}>English</option>
            </select>
        </div>
        <button class="btn" onclick="saveConfig()">保存</button>
        <div id="cfgResult" style="margin-top:10px"></div>
    `;
}

async function saveConfig() {
    const body = {
        maxMemory: parseInt(document.getElementById('maxMem').value) || 0,
        minMemory: parseInt(document.getElementById('minMem').value) || 0,
        threads: parseInt(document.getElementById('threads').value) || 0,
        width: parseInt(document.getElementById('width').value) || 0,
        height: parseInt(document.getElementById('height').value) || 0,
        javaPath: document.getElementById('javaPath').value,
        lang: document.getElementById('langSel').value
    };
    const r = await API.config.set(body);
    document.getElementById('cfgResult').textContent = r.ok ? '配置已保存' : `保存失败: ${errText(r)}`;
}

// ---------- 文档 ----------

async function renderDocs() {
    const apiDoc = await API.docs.api();
    app.innerHTML = `
        <h2>${t('apiDocs')}</h2>
        <div class="api-docs">${apiDoc}</div>
    `;
}

render();
