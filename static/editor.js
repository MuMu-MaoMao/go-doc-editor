// ---------- 前端逻辑 ----------
import { getToken, removeToken, isLoggedIn, redirectToLogin } from '/static/auth.js';

if (!isLoggedIn()) {
    redirectToLogin();
}

let currentFile = null;
let fileList = [];

const fileListEl = document.getElementById('fileList');
const editor = document.getElementById('editor');
const currentFileSpan = document.getElementById('currentFile');
const statusSpan = document.getElementById('statusMsg');
const refreshBtn = document.getElementById('refreshBtn');
const createBtn = document.getElementById('createBtn');
const saveBtn = document.getElementById('saveBtn');
const deleteCurrentBtn = document.getElementById('deleteCurrentBtn');
const newFileNameInput = document.getElementById('newFileName');

function setStatus(msg, isError = false) {
    statusSpan.textContent = msg;
    statusSpan.style.color = isError ? '#b91c1c' : '#555555';
    setTimeout(() => {
        if (statusSpan.textContent === msg && !isError) statusSpan.style.color = '#555555';
    }, 2500);
}

function authHeaders(extra = {}) {
    return {
        ...extra,
        'Authorization': `Bearer ${getToken()}`
    };
}

// 检查是否为 401 未授权响应，若是则清除 token 并跳转到登录页
async function checkAuthResponse(response) {
    if (response.status === 401) {
        removeToken();
        // 读取响应体中的错误信息（现在 middleware 返回 JSON 格式）
        try {
            const data = await response.json();
            console.error('认证失败:', data.error || 'token 已过期');
        } catch (e) {
            console.error('认证失败: token 已过期');
        }
        redirectToLogin();
        return false;
    }
    return true;
}

async function fetchFileList() {
    try {
        const res = await fetch('/api/files', { headers: authHeaders() });
        if (!await checkAuthResponse(res)) return [];
        const data = await res.json();
        if (data.success) {
            fileList = data.files || [];
            renderFileList();
            return fileList;
        } else {
            throw new Error(data.error || '未知错误');
        }
    } catch (err) {
        setStatus('加载文件列表失败: ' + err.message, true);
        fileListEl.innerHTML = '<li style="justify-content:center;">加载失败，请刷新</li>';
        return [];
    }
}

function renderFileList() {
    if (!fileList.length) {
        fileListEl.innerHTML = '<li style="justify-content:center;">📭 暂无文档，新建一个吧</li>';
        return;
    }
    const fragment = document.createDocumentFragment();
    fileList.forEach(filename => {
        const li = document.createElement('li');
        li.className = (currentFile === filename) ? 'active' : '';
        const nameSpan = document.createElement('span');
        nameSpan.className = 'file-name';
        nameSpan.textContent = filename;
        const delBtn = document.createElement('button');
        delBtn.textContent = '✖';
        delBtn.className = 'del-btn';
        delBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            deleteFile(filename);
        });
        li.appendChild(nameSpan);
        li.appendChild(delBtn);
        li.addEventListener('click', () => openFile(filename));
        fragment.appendChild(li);
    });
    fileListEl.innerHTML = '';
    fileListEl.appendChild(fragment);
}

async function openFile(filename) {
    if (!filename) return;
    setStatus(`正在加载 ${filename} ...`);
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(filename)}`, { headers: authHeaders() });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            editor.value = data.content;
            currentFile = filename;
            currentFileSpan.textContent = `📄 ${filename}`;
            setStatus(`已打开 ${filename}`);
            renderFileList();
        } else {
            throw new Error(data.error || '读取失败');
        }
    } catch (err) {
        setStatus(`打开失败: ${err.message}`, true);
        if (err.message.includes('不存在')) {
            await fetchFileList();
            if (currentFile === filename) {
                editor.value = '';
                currentFile = null;
                currentFileSpan.textContent = '未选择文件';
            }
        }
    }
}

async function saveCurrent() {
    if (!currentFile) {
        setStatus('请先选择或新建一个文档', true);
        return;
    }
    const content = editor.value;
    setStatus(`保存中...`);
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(currentFile)}`, {
            method: 'POST',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ content: content })
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`✅ 已保存 ${currentFile}`);
            await fetchFileList();
        } else {
            throw new Error(data.error || '保存失败');
        }
    } catch (err) {
        setStatus(`保存失败: ${err.message}`, true);
    }
}

async function deleteFile(filename) {
    if (!confirm(`确定删除 "${filename}" 吗？不可恢复。`)) return;
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(filename)}`, {
            method: 'DELETE',
            headers: authHeaders()
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`已删除 ${filename}`);
            if (currentFile === filename) {
                editor.value = '';
                currentFile = null;
                currentFileSpan.textContent = '未选择文件';
            }
            await fetchFileList();
        } else {
            throw new Error(data.error || '删除失败');
        }
    } catch (err) {
        setStatus(`删除失败: ${err.message}`, true);
    }
}

async function deleteCurrent() {
    if (!currentFile) {
        setStatus('没有打开任何文件', true);
        return;
    }
    await deleteFile(currentFile);
}

async function createDocument() {
    let newName = newFileNameInput.value.trim();
    if (!newName) {
        setStatus('请输入文件名', true);
        return;
    }
    if (newName.includes('/') || newName.includes('\\') || newName.includes('..')) {
        setStatus('文件名不能包含 / \\ 或 ..', true);
        return;
    }
    if (fileList.includes(newName) && !confirm(`文件 ${newName} 已存在，是否覆盖？`)) {
        return;
    }
    setStatus(`正在创建 ${newName} ...`);
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(newName)}`, {
            method: 'POST',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ content: '' })
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`创建成功，正在打开...`);
            newFileNameInput.value = '';
            await fetchFileList();
            await openFile(newName);
        } else {
            throw new Error(data.error || '创建失败');
        }
    } catch (err) {
        setStatus(`创建失败: ${err.message}`, true);
    }
}

refreshBtn.addEventListener('click', () => fetchFileList().then(() => setStatus('列表已刷新')));
createBtn.addEventListener('click', createDocument);
saveBtn.addEventListener('click', saveCurrent);
deleteCurrentBtn.addEventListener('click', deleteCurrent);
newFileNameInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') createDocument();
});

// ====== AI 集成：打开 AI 面板并附带文档 ======

document.getElementById('aiSendToChatBtn')?.addEventListener('click', () => {
    if (!editor.value) {
        setStatus('⚠️ 编辑器为空，无法附带文档', true);
        return;
    }
    // 只打开 AI 面板，并开启"附带文档"状态，不自动发送消息
    const panel = document.getElementById('aiChatPanel');
    const toggle = document.getElementById('aiChatToggle');
    if (panel && toggle && !panel.classList.contains('visible')) {
        toggle.click();
    }
    // 激活"附带文档"
    const attachBtn = document.getElementById('aiChatAttachDoc');
    if (attachBtn && !attachBtn.classList.contains('active')) {
        attachBtn.click();
    }
    setStatus('📎 AI 面板已打开，附带当前文档');
});

fetchFileList().then(() => setStatus('就绪，点击文档开始编辑'));