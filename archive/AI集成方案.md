> \> **📜 历史文档归档**
> 本文档记录的是 AI 功能集成阶段的原始设计方案和代码。
> 该设计已被整合到规范体系中的 **ADR-002**。
> 请查阅：[docs/adr/ADR-002-deepseek-proxy.md](./docs/adr/ADR-002-deepseek-proxy.md)
> 新功能开发请遵循：[docs/12-功能开发流程规范.md](./docs/12-功能开发流程规范.md)
> *本文档保留作为历史参考。*

方案已按您的反馈重写为 `c:/go-doc-editor/AI集成方案.md`（v2 版本），核心变更如下：

**架构变更**：纯前端调用 → **后端代理模式**
```
前端 → POST /api/ai/chat → 后端 Go → DeepSeek API → 后端 SSE 逐块转发 → 前端展示
```

**新增后端文件（2个）**：
- `internal/service/ai_service.go` — 调用 DeepSeek 流式 API，通过回调逐块返回
- `internal/handler/ai_handler.go` — 解析前端请求，以 SSE (`text/event-stream`) 格式流式返回

**修改后端文件（1个）**：
- `cmd/server/main.go` — 创建 `aiService` + `aiHandler`，注册 `/api/ai/chat` 路由（受 JWT 中间件保护）

**新增前端文件（1个）**：
- `static/ai-chat.js` — 前端对话面板 UI + ReadableStream 接收 SSE + 导出 `sendDocumentToAI()`

**修改前端文件（3个）**：
- `static/editor.html` — 添加 AI 面板 HTML、浮动按钮 💬、工具栏"📤 问 AI"按钮
- `static/editor.js` — 导入 `sendDocumentToAI`，绑定按钮事件
- `static/style.css` — AI 面板全套样式

**密钥**保存在后端 `cmd/server/main.go` 中，前端不直接暴露。

# AI 对话集成方案 v2 — DeepSeek-Flash（后端代理）

## 概述

在本地文档编辑器的编辑页面（`editor.html`）中，增加一个 **AI 对话功能**。用户选中或编辑文档时，可点击工具栏按钮，将当前文档内容 + 自定义提示词发送到后端；后端 Go 服务负责调用 DeepSeek-Flash API，将响应流式（SSE）返回给前端展示。

**核心变化**：API 密钥保存在后端，前端不直接接触密钥；请求链路为 `前端 → 后端 Go → DeepSeek API → 后端 Go → 前端 SSE`。

---

## 目标与边界

| 需求 | 说明 |
|------|------|
| 模型 | `deepseek-chat` (DeepSeek-Flash) |
| API 密钥 |  |
| 请求链路 | 前端 → 后端 Go API → DeepSeek API → 后端流式返回 → 前端 SSE |
| 集成范围 | **前端 + 后端**均需修改 |
| 新增文件 | 3 个：`internal/service/ai_service.go`、`internal/handler/ai_handler.go`、`static/ai-chat.js` |
| 修改文件 | 3 个：`cmd/server/main.go`、`static/editor.html`、`static/style.css` |

---

## 架构设计

```
                   ┌──────────────────────┐
                   │   前端 editor.html    │
                   │   (ai-chat.js)        │
                   └──────┬───────────────┘
                          │ POST /api/ai/chat  (JSON: {prompt, content})
                          │ 响应：SSE (text/event-stream)
                          ▼
┌──────────────────────────────────────────────┐
│          后端 Go (main.go)                    │
│                                              │
│  POST /api/ai/chat                           │
│    → handler/ai_handler.go                   │
│        → service/ai_service.go               │
│            → POST https://api.deepseek.com   │
│                /v1/chat/completions (stream)  │
│            ← SSE 逐块转发到前端               │
└──────────────────────────────────────────────┘
```

---

## 详细修改清单

### 1. 新增 `internal/service/ai_service.go` — AI 服务层

```go
package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const deepseekAPIURL = "https://api.deepseek.com/v1/chat/completions"

type AIService struct {
	apiKey string
}

func NewAIService(apiKey string) *AIService {
	return &AIService{apiKey: apiKey}
}

// deepseek 请求体
type deepseekRequest struct {
	Model       string          `json:"model"`
	Messages    []deepseekMsg   `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type deepseekMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// deepseek 流式响应中的 data 块
type deepseekStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// 调用 DeepSeek 流式 API，将结果通过回调函数逐块返回
// onChunk: 每次收到内容片段时调用
// 返回完整内容（用于记录历史）
func (s *AIService) ChatStream(prompt, documentContent string, onChunk func(text string)) (string, error) {
	// 构建消息
	messages := []deepseekMsg{
		{
			Role:    "system",
			Content: "你是一个专业的文档编辑助手。帮助用户编辑、润色、翻译、总结文档内容。请用中文回复。",
		},
	}

	// 如果有文档内容，先发送文档上下文
	if documentContent != "" {
		messages = append(messages, deepseekMsg{
			Role:    "user",
			Content: fmt.Sprintf("以下是我的文档内容：\n\n```\n%s\n```", documentContent),
		})
	}

	// 用户的具体指令
	messages = append(messages, deepseekMsg{
		Role:    "user",
		Content: prompt,
	})

	reqBody := deepseekRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	req, err := http.NewRequest("POST", deepseekAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 DeepSeek API 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("DeepSeek API 返回错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	// 读取 SSE 流
	scanner := bufio.NewScanner(resp.Body)
	// 增加 buffer 大小以处理长行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fullContent string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk deepseekStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // 跳过无法解析的块
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				fullContent += delta
				onChunk(delta)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullContent, fmt.Errorf("读取流响应失败: %v", err)
	}

	return fullContent, nil
}

// Chat 非流式版本（备选）
func (s *AIService) Chat(prompt, documentContent string) (string, error) {
	var fullContent string
	_, err := s.ChatStream(prompt, documentContent, func(text string) {
		fullContent += text
	})
	return fullContent, err
}
```

### 2. 新增 `internal/handler/ai_handler.go` — AI HTTP 处理器

```go
package handler

import (
	"encoding/json"
	"fmt"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"log"
	"net/http"
	"time"
)

type AIHandler struct {
	aiService *service.AIService
}

func NewAIHandler(svc *service.AIService) *AIHandler {
	return &AIHandler{aiService: svc}
}

type chatRequest struct {
	Prompt  string `json:"prompt"`  // 用户提示词
	Content string `json:"content"` // 当前文档内容（可选）
}

func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	// 验证登录（中间件已处理）
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"未认证"}`, http.StatusUnauthorized)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求体"}`, http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, `{"error":"提示词不能为空"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[AI] 用户 %s 发起 AI 请求，prompt 长度: %d, 文档内容长度: %d",
		username, len(req.Prompt), len(req.Content))

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"不支持流式传输"}`, http.StatusInternalServerError)
		return
	}

	// 发送开始标记
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"start"}`)
	flusher.Flush()

	// 调用 AI 服务，流式返回
	fullContent, err := h.aiService.ChatStream(req.Prompt, req.Content, func(text string) {
		// 对内容中的换行和特殊字符进行转义，确保 SSE 格式正确
		escaped := escapeSSE(text)
		fmt.Fprintf(w, "data: %s\n\n", escaped)
		flusher.Flush()
	})

	if err != nil {
		log.Printf("[AI] 用户 %s AI 请求失败: %v", username, err)
		// 发送错误事件
		errJSON, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", string(errJSON))
		flusher.Flush()
		return
	}

	// 发送完成标记（附带完整内容用于前端记录历史）
	doneJSON, _ := json.Marshal(map[string]string{
		"type":    "done",
		"content": fullContent,
	})
	fmt.Fprintf(w, "data: %s\n\n", string(doneJSON))
	flusher.Flush()

	log.Printf("[AI] 用户 %s AI 请求完成，总长度: %d", username, len(fullContent))
}

// SSE 转义：将内容包装为 SSE data 格式
func escapeSSE(text string) string {
	// 将多行文本转换为多个 "data:" 行
	// 但这里我们直接发送 JSON 格式，每块只包含纯文本 delta
	escaped, _ := json.Marshal(map[string]string{"type": "chunk", "delta": text})
	return string(escaped)
}
```

### 3. 修改 `cmd/server/main.go` — 注册 AI 路由

在 `main()` 函数中，新增 AI 相关初始化与路由：

```go
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"go-doc-editor/internal/config"
	"go-doc-editor/internal/handler"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"go-doc-editor/internal/user"
)

func main() {
	cfg := config.Load()

	// 用户存储
	dataDir := filepath.Join(filepath.Dir(cfg.StorageDir), "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("无法创建数据目录: %v", err)
	}
	userStore, err := user.NewStore(dataDir)
	if err != nil {
		log.Fatalf("初始化用户存储失败: %v", err)
	}

	fileService := service.NewFileService(cfg.StorageDir)
	// +++ 新增：创建 AI 服务（API 密钥硬编码，后续可改为配置）
	aiService := service.NewAIService("sk-664c572e17fd40d6aacfa476c05c475e")

	authHandler := handler.NewAuthHandler(userStore)
	fileHandler := handler.NewFileHandler(fileService)
	// +++ 新增：创建 AI 处理器
	aiHandler := handler.NewAIHandler(aiService)

	// 公开路由
	http.HandleFunc("/api/register", authHandler.Register)
	http.HandleFunc("/api/login", authHandler.Login)

	// +++ 新增：AI 对话路由（受保护）
	http.HandleFunc("/api/ai/chat", middleware.AuthMiddleware(aiHandler.Chat))

	// 受保护路由（文档操作）
	http.HandleFunc("/api/files", middleware.AuthMiddleware(fileHandler.ListFiles))
	http.HandleFunc("/api/file/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			middleware.AuthMiddleware(fileHandler.ReadFile)(w, r)
		case http.MethodPost:
			middleware.AuthMiddleware(fileHandler.SaveFile)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(fileHandler.DeleteFile)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 静态文件
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		http.ServeFile(w, r, "./static"+r.URL.Path)
	})

	log.Printf("服务器启动，访问 http://localhost%s", cfg.Port)
	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
```

**变更点标注**（4 处）：
1. `import` 新增 `"go-doc-editor/internal/handler"`（已有）— 无需改动
2. 创建 `aiService`：`service.NewAIService("sk-...")`
3. 创建 `aiHandler`：`handler.NewAIHandler(aiService)`
4. 注册路由：`http.HandleFunc("/api/ai/chat", middleware.AuthMiddleware(aiHandler.Chat))`

### 4. 新增 `static/ai-chat.js` — 前端 AI 对话模块

```javascript
// ====== AI Chat Module ======
// 与后端 AI API 交互，不直接调用 DeepSeek
import { getToken } from '/static/auth.js';

// DOM 元素
const chatPanel = document.getElementById('aiChatPanel');
const chatToggle = document.getElementById('aiChatToggle');
const chatClose = document.getElementById('aiChatClose');
const chatMessages = document.getElementById('aiChatMessages');
const chatInput = document.getElementById('aiChatInput');
const chatSend = document.getElementById('aiChatSend');
const chatClear = document.getElementById('aiChatClear');

// 对话历史（仅用于前端展示）
let fullResponses = [];

// 切换面板可见性
chatToggle.addEventListener('click', () => {
    chatPanel.classList.toggle('visible');
    if (chatPanel.classList.contains('visible')) {
        chatInput.focus();
    }
});
chatClose.addEventListener('click', () => {
    chatPanel.classList.remove('visible');
});

// 发送消息
async function sendMessage(prompt, documentContent = '') {
    if (!prompt.trim()) return;

    // 禁用输入
    chatInput.disabled = true;
    chatSend.disabled = true;

    // 显示用户消息
    appendMessage('user', prompt);
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
                prompt: prompt,
                content: documentContent
            })
        });

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
                            contentSpan.textContent = fullContent;
                            chatMessages.scrollTop = chatMessages.scrollHeight;
                        } else if (data.type === 'done') {
                            fullContent = data.content || fullContent;
                            contentSpan.textContent = fullContent;
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

        // 记录完整响应
        fullResponses.push({ prompt, response: fullContent });

    } catch (err) {
        contentSpan.textContent = `❌ ${err.message}`;
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
    div.innerHTML = `
        <div class="chat-msg-avatar">${role === 'user' ? '👤' : '🤖'}</div>
        <div class="chat-msg-content">${escapeHtml(text)}</div>
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
chatClear.addEventListener('click', () => {
    fullResponses = [];
    chatMessages.innerHTML = '';
    appendMessage('system', '💬 对话已清空。');
});

// 发送事件
chatSend.addEventListener('click', () => {
    sendMessage(chatInput.value);
});
chatInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage(chatInput.value);
    }
});

// 初始化欢迎消息
appendMessage('system', '👋 你好！我是 AI 编辑助手（DeepSeek-Flash）。\n• 在输入框提问，或点击工具栏"📤 问 AI"发送当前文档\n• 可以帮你润色、翻译、续写、总结、问答');

// 导出供 editor.js 使用
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

    // 自动构建提示词
    const fullPrompt = `${prompt}\n\n\`\`\`\n${content}\n\`\`\``;
    return sendMessage(fullPrompt);
}
```

### 5. 修改 `static/editor.html` — 添加 AI 面板与按钮

在 `</div>`（主布局结束）后、`<script>` 之前，添加 AI 面板的 HTML：

```html
<!-- ====== AI 对话面板 ====== -->
<button id="aiChatToggle" class="ai-toggle-btn" title="AI 助手">💬</button>

<div id="aiChatPanel" class="ai-chat-panel">
    <div class="ai-chat-header">
        <span>🤖 AI 助手 (DeepSeek-Flash)</span>
        <div class="ai-chat-actions">
            <button id="aiChatClear" title="清空对话">🗑</button>
            <button id="aiChatClose" title="关闭">✕</button>
        </div>
    </div>
    <div class="ai-chat-messages" id="aiChatMessages"></div>
    <div class="ai-chat-input-area">
        <textarea id="aiChatInput" placeholder="输入你的问题..." rows="3"></textarea>
        <button id="aiChatSend">发送</button>
    </div>
</div>
```

在 `<div class="action-buttons">` 中添加按钮：

```html
<button id="aiSendToChatBtn" title="将当前文档内容发送给 AI 处理">📤 问 AI</button>
```

添加脚本引入：

```html
<script type="module" src="/static/auth.js"></script>
<script type="module" src="/static/editor.js"></script>
<script type="module" src="/static/ai-chat.js"></script>
```

### 6. 修改 `static/editor.js` — 添加"📤 问 AI"按钮事件

在文件末尾添加：

```javascript
// ====== AI 集成：发送文档到 AI ======
import { sendDocumentToAI } from '/static/ai-chat.js';

document.getElementById('aiSendToChatBtn')?.addEventListener('click', async () => {
    if (!editor.value) {
        setStatus('编辑器为空，无法发送', true);
        return;
    }
    setStatus('正在发送到 AI 助手...');
    try {
        await sendDocumentToAI('请帮我处理以下文档内容：');
        setStatus('✅ AI 响应完成');
    } catch (err) {
        setStatus('发送失败: ' + err.message, true);
    }
});
```

### 7. 修改 `static/style.css` — 添加 AI 面板样式

在文件末尾追加 AI 面板样式（CSS 代码与 v1 方案一致，见本文底部*附录*）。

---

## 文件变更汇总

| 操作 | 文件 | 说明 |
|------|------|------|
| **新增** | `internal/service/ai_service.go` | AI 服务层：调用 DeepSeek 流式 API，通过回调逐块返回 |
| **新增** | `internal/handler/ai_handler.go` | AI HTTP 处理器：解析请求、SSE 响应、流式转发 |
| **新增** | `static/ai-chat.js` | 前端 AI 对话模块：SSE 接收、UI 交互、导出 API |
| **修改** | `cmd/server/main.go` | 创建 AI 服务和处理器，注册 `/api/ai/chat` 路由 |
| **修改** | `static/editor.html` | 添加 AI 面板、浮动按钮、"📤 问 AI"按钮、脚本引入 |
| **修改** | `static/editor.js` | 添加"📤 问 AI"按钮事件 |
| **修改** | `static/style.css` | 追加 AI 面板样式 |

---

## 交互流程

```
1. 用户编辑文档 → 点击工具栏 "📤 问 AI"
2. 前端读取 editor.value + 提示词，POST → /api/ai/chat
3. 后端 handler 解析 JSON → 调用 aiService.ChatStream()
4. aiService 向 api.deepseek.com 发起流式 POST
5. 后端读取 SSE → 通过 w.(http.Flusher) 逐块转发给前端
6. 前端 ReadableStream 接收 SSE → 实时更新对话气泡
7. 完成时后端发送 {type:"done"}，前端记录完整响应
```

---

## 注意事项

1. **API 密钥**：当前硬编码在 `cmd/server/main.go` 中，后续可移至 config 配置或环境变量。
2. **CORS**：后端到 DeepSeek API 是服务器间调用，无 CORS 问题。
3. **SSE 兼容性**：使用标准 `text/event-stream` + `ReadableStream`，兼容现代浏览器。
4. **超时处理**：建议后续增加请求超时控制（`context.WithTimeout`）。
5. **错误处理**：DeepSeek API 错误会通过 SSE `{type:"error"}` 事件通知前端。

---

## 附录：AI 面板 CSS 样式

```css
/* ========== AI 对话面板 ========== */

.ai-toggle-btn {
    position: fixed;
    bottom: 28px;
    right: 28px;
    z-index: 1000;
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: #1a1a1a;
    color: white;
    border: none;
    font-size: 1.3rem;
    cursor: pointer;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
}

.ai-toggle-btn:hover {
    background: #000;
    transform: scale(1.05);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.3);
}

.ai-chat-panel {
    position: fixed;
    bottom: 100px;
    right: 28px;
    z-index: 999;
    width: 380px;
    height: 560px;
    max-height: 70vh;
    background: white;
    border-radius: 20px;
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.18);
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    transform: translateY(20px);
    opacity: 0;
    visibility: hidden;
    transition: all 0.25s ease;
    overflow: hidden;
}

.ai-chat-panel.visible {
    transform: translateY(0);
    opacity: 1;
    visibility: visible;
}

.ai-chat-header {
    background: linear-gradient(135deg, #1a1a1a 0%, #2c2c2c 100%);
    color: white;
    padding: 14px 18px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.9rem;
    font-weight: 500;
}

.ai-chat-actions button {
    background: transparent;
    border: none;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    font-size: 1rem;
    padding: 4px 8px;
    border-radius: 12px;
    transition: all 0.2s;
}

.ai-chat-actions button:hover {
    background: rgba(255, 255, 255, 0.15);
    color: white;
}

.ai-chat-messages {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    background: #fafafa;
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.chat-message {
    display: flex;
    gap: 10px;
    max-width: 90%;
    animation: fadeIn 0.2s ease;
}

.chat-message.user {
    align-self: flex-end;
    flex-direction: row-reverse;
}

.chat-message.system {
    align-self: center;
    max-width: 100%;
    font-size: 0.82rem;
    color: var(--text-muted);
    text-align: center;
    background: #f0f0f0;
    padding: 10px 14px;
    border-radius: 14px;
    white-space: pre-wrap;
    line-height: 1.5;
}

.chat-message.assistant {
    align-self: flex-start;
}

.chat-msg-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: #e8e8e8;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1rem;
    flex-shrink: 0;
}

.chat-message.user .chat-msg-avatar {
    background: #1a1a1a;
}

.chat-msg-content {
    background: white;
    padding: 10px 14px;
    border-radius: 16px;
    font-size: 0.85rem;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-word;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
    color: var(--text-primary);
}

.chat-message.user .chat-msg-content {
    background: #1a1a1a;
    color: white;
    border-bottom-right-radius: 4px;
}

.chat-message.assistant .chat-msg-content {
    border-bottom-left-radius: 4px;
}

.ai-chat-input-area {
    border-top: 1px solid var(--border);
    padding: 12px;
    display: flex;
    gap: 8px;
    background: white;
    align-items: flex-end;
}

.ai-chat-input-area textarea {
    flex: 1;
    padding: 10px 14px;
    border: 1px solid var(--border);
    border-radius: 14px;
    font-size: 0.85rem;
    font-family: inherit;
    resize: none;
    transition: all 0.2s;
    background: #fafafa;
}

.ai-chat-input-area textarea:focus {
    outline: none;
    border-color: #1a1a1a;
    background: white;
}

.ai-chat-input-area button {
    padding: 10px 20px;
    background: #1a1a1a;
    color: white;
    border: none;
    border-radius: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
}

.ai-chat-input-area button:hover {
    background: #000;
}

.ai-chat-input-area button:disabled {
    background: #aaa;
    cursor: not-allowed;
}

@keyframes fadeIn {
    from { opacity: 0; transform: translateY(4px); }
    to   { opacity: 1; transform: translateY(0); }
}

@media (max-width: 680px) {
    .ai-chat-panel {
        right: 12px;
        left: 12px;
        width: auto;
        bottom: 90px;
        max-height: 60vh;
    }
    .ai-toggle-btn {
        bottom: 20px;
        right: 20px;
        width: 48px;
        height: 48px;
        font-size: 1.1rem;
    }
}