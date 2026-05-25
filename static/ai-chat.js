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

// 附带文档 toggle 状态
let attachDocument = false;

// 初始化时添加 system 消息
messages.push({
    role: 'system',
    content: '你是一个文档编辑助手。日常帮助用户编辑、润色、翻译、总结文档内容。请用中文回复。'
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

// 初始化欢迎消息
appendMessage('system', '👋 你好！我是 AI 编辑助手 (DeepSeek-Flash)。\n• 在输入框提问，或点击工具栏"📤 问 AI"发送当前文档\n• 可以帮你润色、翻译、续写、总结、问答\n• 点击"📎 附带文档"可将当前文档内容作为上下文发送');

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
