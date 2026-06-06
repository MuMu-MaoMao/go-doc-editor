// ====== 用户主页模块 ======
import { getToken, removeToken, redirectToLogin, isLoggedIn } from '/static/auth.js';

if (!isLoggedIn()) {
    redirectToLogin();
}

const profileApp = document.getElementById('profileApp');

function formatTime(timeStr) {
    if (!timeStr) return '暂无记录';
    return timeStr;
}

async function loadProfile() {
    try {
        const res = await fetch('/api/user/profile', {
            headers: { 'Authorization': `Bearer ${getToken()}` }
        });

        if (res.status === 401) {
            removeToken();
            redirectToLogin();
            return;
        }

        const data = await res.json();
        if (!data.success) {
            throw new Error(data.error || '加载失败');
        }

        renderProfile(data.data);
    } catch (err) {
        profileApp.innerHTML = `<div class="error-msg">❌ ${err.message}</div>`;
    }
}

function renderProfile(data) {
    const loginCount = data.loginLogs ? data.loginLogs.length : 0;
    const aiKeys = data.aiKeys || [];

    profileApp.innerHTML = `
        <div class="profile-container">
            <div class="profile-header">
                <div class="profile-avatar">👤</div>
                <div class="profile-info">
                    <h2>${escapeHtml(data.username)}</h2>
                    <div class="meta">共 ${loginCount} 次登录记录</div>
                </div>
            </div>

            <div class="profile-section">
                <h3>账户信息</h3>
                <div class="info-row">
                    <span class="label">用户名</span>
                    <span class="value">${escapeHtml(data.username)}</span>
                </div>
                <div class="info-row">
                    <span class="label">注册时间</span>
                    <span class="value">${formatTime(data.createdAt)}</span>
                </div>
                <div class="info-row">
                    <span class="label">最近登录</span>
                    <span class="value">${formatTime(data.lastLoginAt)}</span>
                </div>
            </div>

            <div class="profile-section">
                <h3>登录历史</h3>
                ${loginCount === 0
                    ? '<div class="info-row"><span class="label" style="color:var(--text-muted)">暂无登录记录</span></div>'
                    : `<ul class="login-list">${data.loginLogs.map((log, i) => `
                        <li>
                            <span><span class="login-index">#${loginCount - i}</span> ${formatTime(log.loginTime)}</span>
                        </li>
                    `).join('')}</ul>`
                }
            </div>

            <div class="profile-section" id="aiKeySection">
                <h3>🤖 AI 配置 <span style="font-weight:normal;font-size:0.75rem;color:var(--text-muted)">（在此管理 AI-Key 和模型）</span></h3>
                <div id="aiKeyList"></div>
                <div class="ai-key-form" id="aiKeyForm" style="margin-top:12px;"></div>
                <button id="showAddKeyBtn" class="back-link" style="margin-top:8px;">＋ 添加 AI Key</button>
            </div>

            <a href="/" class="back-link">← 返回首页</a>
            <a href="/editor.html" class="back-link" style="margin-left:8px;">📝 进入编辑器</a>
        </div>
    `;

    // 渲染 AI Key 列表
    renderAIKeyList(aiKeys);

    // 绑定添加按钮
    document.getElementById('showAddKeyBtn')?.addEventListener('click', () => {
        showAddKeyForm();
    });
}

// ====== AI Key 管理 ======

function renderAIKeyList(keys) {
    const container = document.getElementById('aiKeyList');
    if (!container) return;

    if (keys.length === 0) {
        container.innerHTML = '<div class="info-row"><span class="label" style="color:var(--text-muted)">尚未配置 AI Key，添加后即可使用自己的 API Key</span></div>';
        return;
    }

    container.innerHTML = keys.map(k => `
        <div class="ai-key-item ${k.isActive ? 'active' : ''}" data-id="${k.id}">
            <div class="ai-key-info">
                <div class="ai-key-name">
                    ${k.isActive ? '✅' : '○'} ${escapeHtml(k.keyName)}
                </div>
                <div class="ai-key-detail">
                    模型：${k.model === 'deepseek-v4-flash' || k.model === 'deepseek-chat' ? 'DeepSeek-Flash（快速）' : k.model === 'deepseek-v4-pro' || k.model === 'deepseek-reasoner' ? 'DeepSeek-Pro（推理）' : escapeHtml(k.model)}
                    &nbsp;|&nbsp; ${escapeHtml(k.apiUrl)}
                </div>
                <div class="ai-key-detail">添加于 ${formatTime(k.createdAt)}</div>
            </div>
            <div class="ai-key-actions">
                ${!k.isActive ? `<button class="key-btn activate-btn" data-id="${k.id}">激活</button>` : '<span class="key-badge">当前使用</span>'}
                <button class="key-btn delete-btn" data-id="${k.id}">删除</button>
            </div>
        </div>
    `).join('');

    // 绑定激活按钮
    container.querySelectorAll('.activate-btn').forEach(btn => {
        btn.addEventListener('click', () => activateKey(parseInt(btn.dataset.id)));
    });
    // 绑定删除按钮
    container.querySelectorAll('.delete-btn').forEach(btn => {
        btn.addEventListener('click', () => deleteKey(parseInt(btn.dataset.id)));
    });
}

function showAddKeyForm() {
    const formContainer = document.getElementById('aiKeyForm');
    if (!formContainer) return;

    formContainer.innerHTML = `
        <div class="ai-key-form-inner">
            <input type="text" id="newKeyName" placeholder="名称，如"我的 DeepSeek"" class="key-input">
            <input type="password" id="newApiKey" placeholder="API Key" class="key-input">
            <select id="newModel" class="key-input key-select">
                <option value="deepseek-v4-flash">DeepSeek-Flash（快速）</option>
                <option value="deepseek-v4-pro">DeepSeek-Pro（推理）</option>
            </select>
            <input type="text" id="newApiUrl" value="https://api.deepseek.com/chat/completions" class="key-input">
            <div class="ai-key-form-actions">
                <button id="saveKeyBtn" class="back-link" style="cursor:pointer;">保存</button>
                <button id="cancelAddKeyBtn" class="back-link" style="cursor:pointer;">取消</button>
            </div>
        </div>
    `;

    document.getElementById('saveKeyBtn')?.addEventListener('click', saveNewKey);
    document.getElementById('cancelAddKeyBtn')?.addEventListener('click', () => {
        formContainer.innerHTML = '';
    });
}

async function saveNewKey() {
    const keyName = document.getElementById('newKeyName')?.value.trim();
    const apiKey = document.getElementById('newApiKey')?.value.trim();
    const modelSelect = document.getElementById('newModel');
    const model = modelSelect?.value || 'deepseek-v4-flash';
    const apiUrl = document.getElementById('newApiUrl')?.value.trim() || 'https://api.deepseek.com/chat/completions';

    if (!keyName || !apiKey) {
        alert('名称和 API Key 不能为空');
        return;
    }

    try {
        const res = await fetch('/api/user/ai-keys', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${getToken()}`
            },
            body: JSON.stringify({ keyName, apiKey, apiUrl, model })
        });
        const data = await res.json();
        if (data.success) {
            // 刷新整个页面以更新列表
            loadProfile();
        } else {
            alert('添加失败: ' + (data.error || '未知错误'));
        }
    } catch (err) {
        alert('添加失败: ' + err.message);
    }
}

async function activateKey(id) {
    try {
        const res = await fetch(`/api/user/ai-keys/${id}/activate`, {
            method: 'PUT',
            headers: { 'Authorization': `Bearer ${getToken()}` }
        });
        const data = await res.json();
        if (data.success) {
            loadProfile();
        } else {
            alert('激活失败: ' + (data.error || '未知错误'));
        }
    } catch (err) {
        alert('激活失败: ' + err.message);
    }
}

async function deleteKey(id) {
    if (!confirm('确定删除这个 AI Key 吗？')) return;
    try {
        const res = await fetch(`/api/user/ai-keys/${id}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${getToken()}` }
        });
        const data = await res.json();
        if (data.success) {
            loadProfile();
        } else {
            alert('删除失败: ' + (data.error || '未知错误'));
        }
    } catch (err) {
        alert('删除失败: ' + err.message);
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

loadProfile();
