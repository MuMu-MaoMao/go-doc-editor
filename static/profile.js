// ====== 用户主页模块 ======
import { getToken, removeToken, redirectToLogin, isLoggedIn } from '/static/auth.js';

if (!isLoggedIn()) {
    redirectToLogin();
}

const profileApp = document.getElementById('profileApp');

// 格式化时间
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

            <a href="/" class="back-link">← 返回首页</a>
            <a href="/editor.html" class="back-link" style="margin-left:8px;">📝 进入编辑器</a>
        </div>
    `;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

loadProfile();
