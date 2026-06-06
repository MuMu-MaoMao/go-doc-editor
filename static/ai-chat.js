// ====== AI Chat Module ======
// 与后端 AI API 交互，不直接调用 DeepSeek
import { getToken, removeToken, redirectToLogin } from '/static/auth.js';

// DOM 元素
const chatPanel = document.getElementById('aiChatPanel');
const chatToggle = document.getElementById('aiChatToggle');
const chatClose = document.getElementById('aiChatClose');
const chatMessages = document.getElementById('aiChatMessages');
const chatInput = document.getElementById('aiChatInput');
const chatSend = document.getElementById('aiChatSend');
const chatClear = document.getElementById('aiChatClear');
const chatAttachDoc = document.getElementById('aiChatAttachDoc');

// 对话历史（包含完整消息列表，用于发送给后端）
let messages = [];

// 当前角色状态
let currentRoleId = null;        // 当前选中的角色 ID
let roles = [];                  // 从后端获取的角色列表

// 角色下拉菜单状态
let roleDropdownOpen = false;

// 拖拽状态
let isDragging = false;
let dragStartX = 0, dragStartY = 0;
let dragPanelLeft = 0, dragPanelTop = 0;

// 拉伸状态
let isResizing = false;
let resizeDirection = '';
let resizeStartX = 0, resizeStartY = 0;
let resizeStartW = 0, resizeStartH = 0;
let resizeStartL = 0, resizeStartT = 0;

// 附带文档 toggle 状态
let attachDocument = false;

// 从后端获取角色列表
async function fetchRoles() {
    try {
        const res = await fetch('/api/ai/roles');
        const data = await res.json();
        if (data.success && data.roles && data.roles.length > 0) {
            roles = data.roles;
            // 默认不使用角色扮演（通用助手）
            currentRoleId = null;
            messages = [{ role: 'system', content: DEFAULT_SYSTEM_PROMPT }];
            // 渲染角色下拉菜单
            renderRoleDropdown();
            updateRoleButtonText();
        }
    } catch (err) {
        console.error('获取角色列表失败:', err);
        // 保底：使用默认 system message
        messages.push({
            role: 'system',
            content: '你是一个文档编辑助手。日常帮助用户编辑、润色、翻译、总结文档内容。请用中文回复。'
        });
    }
}

// 默认的通用 system prompt（未选择任何角色时使用）
const DEFAULT_SYSTEM_PROMPT = '你是一个文档编辑助手。日常帮助用户编辑、润色、翻译、总结文档内容。请用中文回复。';

// 切换到指定角色（传 null 表示不使用角色扮演，恢复默认助手）
function switchToRole(roleId) {
    currentRoleId = roleId;
    const role = roleId ? roles.find(r => r.id === roleId) : null;
    // 重置消息历史，第一条为 system prompt
    messages = [{
        role: 'system',
        content: role ? role.systemPrompt : DEFAULT_SYSTEM_PROMPT
    }];
    // 更新角色按钮文本
    updateRoleButtonText();
}

// 获取角色的短 emoji 标识
function getRoleEmoji(roleId) {
    const emojiMap = {
        'professional-editor': '👔',
        'humorous-writer': '😄',
        'strict-mentor': '📐',
        'friendly-reader': '📖',
        'consultant': '💼'
    };
    return emojiMap[roleId] || '🎭';
}

// 更新角色按钮文字
function updateRoleButtonText() {
    const btn = document.getElementById('aiChatRoleBtn');
    if (!btn) return;
    if (!currentRoleId) {
        btn.textContent = '🎭 不使用 ▼';
        return;
    }
    const role = roles.find(r => r.id === currentRoleId);
    if (role) {
        btn.textContent = `${getRoleEmoji(role.id)} ${role.name} ▼`;
    }
}

// 渲染角色下拉菜单选项
function renderRoleDropdown() {
    const menu = document.getElementById('roleDropdownMenu');
    if (!menu) return;
    menu.innerHTML = '';

    // 第一项："不使用角色扮演"
    const noneItem = document.createElement('div');
    noneItem.className = 'role-dropdown-item' + (!currentRoleId ? ' active' : '');
    noneItem.innerHTML = `
        <span class="role-name">🚫 不使用角色扮演</span>
        <span class="role-desc">恢复为通用 AI 编辑助手</span>
    `;
    noneItem.addEventListener('click', (e) => {
        e.stopPropagation();
        if (!currentRoleId) { closeRoleDropdown(); return; }
        switchToRole(null);
        closeRoleDropdown();
        chatMessages.innerHTML = '';
        appendMessage('system', '🔄 已关闭角色扮演，恢复为通用助手');
    });
    menu.appendChild(noneItem);

    // 分隔线
    const divider = document.createElement('div');
    divider.style.cssText = 'height:1px; background:var(--border); margin:4px 0;';
    menu.appendChild(divider);

    // 各预设角色
    roles.forEach(role => {
        const item = document.createElement('div');
        item.className = 'role-dropdown-item' + (role.id === currentRoleId ? ' active' : '');
        item.dataset.roleId = role.id;
        item.innerHTML = `
            <span class="role-name">${getRoleEmoji(role.id)} ${role.name}</span>
            <span class="role-desc">${role.description}</span>
        `;
        item.addEventListener('click', (e) => {
            e.stopPropagation();
            if (role.id === currentRoleId) {
                closeRoleDropdown();
                return;
            }
            switchToRole(role.id);
            closeRoleDropdown();
            chatMessages.innerHTML = '';
            appendMessage('system', `🔄 已切换到「${role.name}」角色`);
        });
        menu.appendChild(item);
    });
}

// 打开角色下拉菜单
function openRoleDropdown() {
    const menu = document.getElementById('roleDropdownMenu');
    if (!menu) return;
    renderRoleDropdown();
    menu.classList.add('show');
    roleDropdownOpen = true;
}

// 关闭角色下拉菜单
function closeRoleDropdown() {
    const menu = document.getElementById('roleDropdownMenu');
    if (!menu) return;
    menu.classList.remove('show');
    roleDropdownOpen = false;
}

// 切换角色下拉菜单
function toggleRoleDropdown() {
    if (roleDropdownOpen) {
        closeRoleDropdown();
    } else {
        openRoleDropdown();
    }
}

// 初始化时获取角色列表
fetchRoles();

// ====== 面板拖拽功能 ======
const chatHeader = document.querySelector('.ai-chat-header');

// 判断鼠标是否在 header 的按钮区域（不触发拖拽）
function isHeaderButton(el) {
    while (el && el !== chatHeader) {
        if (el.tagName === 'BUTTON') return true;
        el = el.parentElement;
    }
    return false;
}

chatHeader.addEventListener('mousedown', (e) => {
    if (isHeaderButton(e.target)) return;
    if (!chatPanel.classList.contains('visible')) return;
    const rect = chatPanel.getBoundingClientRect();
    chatPanel.style.left = rect.left + 'px';
    chatPanel.style.top = rect.top + 'px';
    chatPanel.style.right = 'auto';
    chatPanel.style.bottom = 'auto';
    dragStartX = e.clientX;
    dragStartY = e.clientY;
    dragPanelLeft = rect.left;
    dragPanelTop = rect.top;
    isDragging = true;
    chatPanel.classList.add('dragging');
    // 拖拽时全局禁止选中文本
    document.body.style.userSelect = 'none';
});

// ====== 面板边框拉伸功能（使用 8 个显式把手） ======

// 开始拉伸：被 resize handle 的 mousedown 调用
function startResize(e, dir) {
    if (!chatPanel.classList.contains('visible')) return;
    const rect = chatPanel.getBoundingClientRect();
    resizeDirection = dir;
    resizeStartX = e.clientX;
    resizeStartY = e.clientY;
    resizeStartW = rect.width;
    resizeStartH = rect.height;
    resizeStartL = rect.left;
    resizeStartT = rect.top;
    // 切换到 left/top 坐标系
    chatPanel.style.left = rect.left + 'px';
    chatPanel.style.top = rect.top + 'px';
    chatPanel.style.right = 'auto';
    chatPanel.style.bottom = 'auto';
    isResizing = true;
    chatPanel.classList.add('dragging');
    // 全局禁止选中文本
    document.body.style.userSelect = 'none';
    document.body.style.cursor = getComputedStyle(e.target).cursor;
    e.preventDefault();
}

// 给所有 resize handle 绑定 mousedown
document.querySelectorAll('.resize-handle').forEach(handle => {
    handle.addEventListener('mousedown', (e) => {
        if (isDragging || isResizing) return;
        startResize(e, handle.dataset.dir);
    });
});

// 全局 mousemove：处理拖拽和拉伸
document.addEventListener('mousemove', (e) => {
    if (isDragging && !isResizing) {
        // 拖拽
        let newLeft = e.clientX - (dragStartX - dragPanelLeft);
        let newTop = e.clientY - (dragStartY - dragPanelTop);
        const maxLeft = window.innerWidth - 100;
        const maxTop = window.innerHeight - 100;
        newLeft = Math.max(-chatPanel.offsetWidth + 100, Math.min(maxLeft, newLeft));
        newTop = Math.max(-20, Math.min(maxTop, newTop));
        chatPanel.style.left = newLeft + 'px';
        chatPanel.style.top = newTop + 'px';
        return;
    }
    if (isResizing) {
        // 拉伸
        let dx = e.clientX - resizeStartX;
        let dy = e.clientY - resizeStartY;
        let newW = resizeStartW, newH = resizeStartH;
        let newL = resizeStartL, newT = resizeStartT;

        if (resizeDirection.includes('e')) newW = Math.max(320, Math.min(750, resizeStartW + dx));
        if (resizeDirection.includes('w')) {
            newW = Math.max(320, Math.min(750, resizeStartW - dx));
            newL = resizeStartL + (resizeStartW - newW);
        }
        if (resizeDirection.includes('s')) newH = Math.max(400, Math.min(window.innerHeight * 0.9, resizeStartH + dy));
        if (resizeDirection.includes('n')) {
            newH = Math.max(400, Math.min(window.innerHeight * 0.9, resizeStartH - dy));
            newT = resizeStartT + (resizeStartH - newH);
        }

        chatPanel.style.width = newW + 'px';
        chatPanel.style.height = newH + 'px';
        chatPanel.style.left = newL + 'px';
        chatPanel.style.top = newT + 'px';
        chatPanel.style.right = 'auto';
        chatPanel.style.bottom = 'auto';
        e.preventDefault();
        return;
    }
});

// 全局 mouseup：停止拖拽和拉伸 + 恢复文本选中
document.addEventListener('mouseup', () => {
    if (isDragging || isResizing) {
        isDragging = false;
        isResizing = false;
        chatPanel.classList.remove('dragging');
        document.body.style.userSelect = '';
        document.body.style.cursor = '';
    }
});

// 切换面板可见性
if (chatToggle) {
    chatToggle.addEventListener('click', () => {
        chatPanel.classList.toggle('visible');
        if (chatPanel.classList.contains('visible') && chatInput) {
            chatInput.focus();
        }
    });
}
if (chatClose) {
    chatClose.addEventListener('click', () => {
        chatPanel.classList.remove('visible');
    });
}

// 切换附带文档按钮状态
if (chatAttachDoc) {
    chatAttachDoc.addEventListener('click', () => {
        attachDocument = !attachDocument;
        chatAttachDoc.textContent = attachDocument ? '📎 已附带文档' : '📎 附带文档';
        chatAttachDoc.classList.toggle('active', attachDocument);
    });
}

// 角色按钮点击事件
const chatRoleBtn = document.getElementById('aiChatRoleBtn');
if (chatRoleBtn) {
    chatRoleBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        toggleRoleDropdown();
    });
}

// 点击页面其他地方关闭角色下拉菜单
document.addEventListener('click', () => {
    if (roleDropdownOpen) closeRoleDropdown();
});

// 发送消息
async function sendMessage(promptText) {
    if (!promptText.trim()) return;

    // 禁用输入
    chatInput.disabled = true;
    chatSend.disabled = true;

    // 构建本次请求的消息列表副本
    const requestMessages = [...messages];

    // 如果开启了"附带文档"，插入文档内容作为上下文
    if (attachDocument) {
        const editor = document.getElementById('editor');
        const content = editor?.value || '';
        if (content) {
            requestMessages.push({
                role: 'user',
                content: `以下是我的文档内容：\n\n\`\`\`\n${content}\n\`\`\``
            });
        }
    }

    // 添加用户消息
    requestMessages.push({ role: 'user', content: promptText });

    // 显示用户消息
    appendMessage('user', promptText);
    chatInput.value = '';

    // 创建助手消息占位
    const assistantDiv = appendMessage('assistant', '');
    const contentSpan = assistantDiv.querySelector('.chat-msg-content');
    let fullContent = '';

    try {
        const response = await fetch('/api/ai/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${getToken()}`
            },
            body: JSON.stringify({
                messages: requestMessages
            })
        });

        // 检查 401 未授权，token 过期则跳转登录
        if (response.status === 401) {
            removeToken();
            try {
                const errData = await response.json().catch(() => ({}));
                console.error('AI 聊天认证失败:', errData.error || 'token 已过期');
            } catch (e) {
                console.error('AI 聊天认证失败: token 已过期');
            }
            redirectToLogin();
            return;
        }

        if (!response.ok) {
            const errData = await response.json().catch(() => ({}));
            throw new Error(errData.error || `HTTP ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop() || '';

            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const jsonStr = line.slice(6).trim();
                    if (!jsonStr) continue;
                    try {
                        const data = JSON.parse(jsonStr);

                        if (data.type === 'start') {
                            // 开始标记，无需处理
                        } else if (data.type === 'chunk') {
                            fullContent += data.delta || '';
                            // 使用 Markdown 渲染助手消息
                            contentSpan.innerHTML = DOMPurify.sanitize(marked.parse(fullContent));
                            chatMessages.scrollTop = chatMessages.scrollHeight;
                        } else if (data.type === 'done') {
                            fullContent = data.content || fullContent;
                            contentSpan.innerHTML = DOMPurify.sanitize(marked.parse(fullContent));
                        } else if (data.type === 'error') {
                            throw new Error(data.message || 'AI 服务错误');
                        }
                    } catch (e) {
                        // 仅当不是 JSON 解析错误时才抛出
                        if (e.message && e.message.startsWith('AI 服务错误')) {
                            throw e;
                        }
                    }
                }
            }
        }

        // 将用户消息和 AI 响应追加到历史记录
        messages.push({ role: 'user', content: promptText });
        messages.push({ role: 'assistant', content: fullContent });

        // 限制消息历史长度（保留最近 20 轮对话，每轮 2 条 + 1 条 system = 最多 41 条）
        if (messages.length > 41) {
            const systemMsg = messages[0];
            messages = [systemMsg, ...messages.slice(-40)];
        }

    } catch (err) {
        contentSpan.innerHTML = DOMPurify.sanitize(`❌ ${err.message}`);
        contentSpan.style.color = '#b91c1c';
    } finally {
        chatInput.disabled = false;
        chatSend.disabled = false;
        chatInput.focus();
    }
}

// 追加消息到对话区域
function appendMessage(role, text) {
    const div = document.createElement('div');
    div.className = `chat-message ${role}`;
    const avatar = role === 'user' ? '👤' : role === 'assistant' ? '🤖' : '💬';
    
    let contentHtml;
    if (role === 'assistant' && text) {
        // AI 消息使用 Markdown 渲染
        contentHtml = DOMPurify.sanitize(marked.parse(text));
    } else if (role === 'user') {
        // 用户消息保持纯文本
        contentHtml = escapeHtml(text);
    } else {
        // system 消息保持纯文本（pre-wrap）
        contentHtml = escapeHtml(text);
    }

    div.innerHTML = `
        <div class="chat-msg-avatar">${avatar}</div>
        <div class="chat-msg-content">${contentHtml}</div>
    `;
    chatMessages.appendChild(div);
    chatMessages.scrollTop = chatMessages.scrollHeight;
    return div;
}

// HTML 转义
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 清空对话
if (chatClear) {
    chatClear.addEventListener('click', () => {
        // 重置消息历史（保留 system 消息）
        messages = [messages[0]];
        attachDocument = false;
        if (chatAttachDoc) {
            chatAttachDoc.textContent = '📎 附带文档';
            chatAttachDoc.classList.remove('active');
        }
        chatMessages.innerHTML = '';
        appendMessage('system', '💬 对话已清空。');
    });
}

// 发送事件
if (chatSend) {
    chatSend.addEventListener('click', () => {
        sendMessage(chatInput.value);
    });
}
if (chatInput) {
    chatInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage(chatInput.value);
        }
    });
}

// 初始化欢迎消息（在角色列表加载完成后显示）
function showWelcomeMessage() {
    const roleName = currentRoleId
        ? (roles.find(r => r.id === currentRoleId)?.name || 'AI 编辑助手')
        : '通用助手（未使用角色扮演）';
    appendMessage('system', `👋 你好！当前：${roleName}\n• 在输入框提问，或点击工具栏"📤 问 AI"发送当前文档\n• 点击工具栏「🎭 角色名」可选择 AI 角色扮演\n• 点击"📎 附带文档"可将当前文档内容作为上下文发送`);
}

// 等待角色加载完成后显示欢迎消息
const checkRolesLoaded = setInterval(() => {
    if (roles.length > 0) {
        clearInterval(checkRolesLoaded);
        showWelcomeMessage();
    }
}, 100);

// 导出供 editor.js 使用 — 启用附带文档并发送 prompt，复用 sendMessage
export function sendDocumentToAI(prompt = '请帮我处理以下文档内容：') {
    const editor = document.getElementById('editor');
    const content = editor?.value || '';
    if (!content) {
        return Promise.reject(new Error('编辑器为空'));
    }

    // 如果面板未打开，先打开
    if (!chatPanel.classList.contains('visible')) {
        chatToggle.click();
    }

    // 开启"附带文档"模式（自动附加当前编辑器内容）
    attachDocument = true;
    if (chatAttachDoc) {
        chatAttachDoc.textContent = '📎 已附带文档';
        chatAttachDoc.classList.add('active');
    }

    // 调用 sendMessage 发送 prompt（sendMessage 内部会根据 attachDocument 自动附带文档）
    sendMessage(prompt);

    // 返回一个 resolved promise（实际结果由界面展示）
    return Promise.resolve();
}
