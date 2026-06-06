> \> **📜 历史文档归档**
> 本文档记录的是 AI 对话优化阶段的原始设计方案。
> 该设计已被整合到规范体系中的 **ADR-003** 和 **ADR-004**。
> 请查阅：[docs/adr/ADR-003-chat-history-management.md](./docs/adr/ADR-003-chat-history-management.md)
> [docs/adr/ADR-004-markdown-rendering.md](./docs/adr/ADR-004-markdown-rendering.md)
> 新功能开发请遵循：[docs/12-功能开发流程规范.md](./docs/12-功能开发流程规范.md)
> *本文档保留作为历史参考。*

# AI 对话功能优化方案

## 概述

本文档针对现有 AI 对话功能的四个痛点提出优化方案，涉及后端（Go）和前端（JavaScript/CSS）的改动。

---

## 优化一：对话历史上下文

### 问题分析

当前 `internal/service/ai_service.go` 中的 `ChatStream()` 方法每次调用都只构建三条消息：

```
system: "你是一个文档编辑助手..."
user:   "以下是我的文档内容：\n\n```\n{documentContent}\n```"
user:   "{prompt}"
```

每次请求都是独立的，后端没有保存历史消息，导致 AI 无法感知之前对话的内容。

### 方案设计

#### 方案一（推荐）：前端维护历史 + 后端透传

**思路**：前端 `ai-chat.js` 维护完整的消息历史数组，每次请求时将历史消息一并发送给后端，后端不再自行构建消息列表，而是直接透传前端提供的消息数组。

**后端改动**：

1. `internal/service/ai_service.go` — `ChatStream` 方法签名变更，接收完整的 `[]deepseekMsg` 消息列表，不再内部构造 system 提示

```go
// 新增：接收完整消息列表
func (s *AIService) ChatStream(messages []deepseekMsg, onChunk func(text string)) (string, error)
```

2. `internal/handler/ai_handler.go` — `chatRequest` 结构体新增 `messages` 字段，替代原来的 `prompt` 和 `content`

```go
type chatRequest struct {
    Messages []deepseekMsg `json:"messages"` // 完整的对话历史（前端维护）
}
```

**前端改动**：

1. `static/ai-chat.js` — 维护 `messages` 数组，存储完整的对话历史（包括 system、user、assistant 消息）

```javascript
// 对话历史（包含完整消息列表，用于发送给后端）
let messages = [];

// 初始化时添加 system 消息
messages.push({
    role: 'system',
    content: '你是一个文档编辑助手。日常帮助用户编辑、润色、翻译、总结文档内容。请用中文回复。'
});
```

2. `sendMessage()` 方法每次将 `messages` 数组发送给后端，收到 AI 响应后把 `{role: 'assistant', content: fullContent}` 追加到 `messages` 中

3. 清空对话时重置 `messages` 数组

4. `sendDocumentToAI()` 不再自动拼接文档内容到 prompt，而是将文档内容作为一条 `user` 消息插入到 `messages` 中

**优点**：
- 后端无状态，逻辑简单
- 前端可控性高，可灵活控制消息列表
- 便于后续支持多轮对话、删除历史消息等交互

**缺点**：
- 消息历史仅存储在前端内存中，刷新页面会丢失


### 选用方案：方案一

（下文的"优化四"中的"附带文档内容"功能会与此方案联动设计。）

---

## 优化二：Markdown 渲染

### 问题分析

当前前端 `appendMessage()` 使用 `textContent`（经 `escapeHtml`）来显示 AI 响应内容，CSS 中 `.chat-msg-content` 使用 `white-space: pre-wrap` 保留换行，但无法渲染 Markdown 格式：

- 代码块 `` ``` `` 无法高亮
- 列表 `-` 无法显示为列表样式
- 加粗 `**`、标题 `#` 等均以纯文本显示

### 方案设计

引入 **marked** 库进行 Markdown 渲染（轻量、无依赖），支持代码高亮可引入 **highlight.js**。

#### 方案一（推荐）：使用 marked 解析 Markdown

**前端改动**：

1. 在 `static/editor.html` 中引入 marked（通过 CDN）：

```html
<!-- Markdown 渲染库 -->
<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
```

或使用 ES Module 方式：

```html
<script type="module">
    import { marked } from 'https://cdn.jsdelivr.net/npm/marked/lib/marked.esm.js';
    window.marked = marked;
</script>
```

2. 修改 `static/ai-chat.js` 中的 `appendMessage()`：

- AI 助手的消息内容使用 `marked.parse(text)` 渲染为 HTML
- 用户的普通文本仍保持纯文本显示
- 注意 XSS 防护：使用 `marked` 的 `sanitize` 配置或 `DOMPurify`（marked 新版本已默认不转义 HTML，需配置或使用 `marked.parse(text, {async: false})`）

3. 修改 CSS 样式，为 Markdown 渲染的内容添加样式：

```css
/* Markdown 渲染内容样式 */
.chat-msg-content h1,
.chat-msg-content h2,
.chat-msg-content h3 { 
    font-size: 1.1em; 
    margin: 8px 0 4px; 
}
.chat-msg-content p { margin: 4px 0; }
.chat-msg-content ul, 
.chat-msg-content ol { 
    padding-left: 20px; 
    margin: 4px 0; 
}
.chat-msg-content code {
    background: #f0f0f0;
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 0.85em;
}
.chat-msg-content pre {
    background: #f5f5f5;
    padding: 12px;
    border-radius: 8px;
    overflow-x: auto;
    margin: 8px 0;
}
.chat-msg-content pre code {
    background: none;
    padding: 0;
}
```

#### 方案二：DOMPurify + marked 双重防护

引入 `DOMPurify` 对 marked 输出的 HTML 进行清洗，防止 XSS 攻击：

```html
<script src="https://cdn.jsdelivr.net/npm/dompurify/dist/purify.min.js"></script>
```

```javascript
const html = marked.parse(text);
const cleanHtml = DOMPurify.sanitize(html);
contentSpan.innerHTML = cleanHtml;
```


## 优化三：AI 面板可拉伸

### 问题分析

当前 AI 面板固定尺寸 `width: 380px; height: 560px`，用户无法根据需要调整面板大小。

### 方案设计

在面板右下角添加一个拉伸手柄（resize handle），使用 CSS `resize` 属性或 JS 拖拽实现。

#### 方案一（推荐）：CSS resize 属性

最简单的方式，在 `.ai-chat-panel` 上添加 CSS resize 属性：

```css
.ai-chat-panel {
    resize: both;        /* 允许水平和垂直拉伸 */
    overflow: auto;      /* resize 需要 overflow 不为 visible */
    min-width: 300px;
    min-height: 400px;
    max-width: 800px;
    max-height: 90vh;
}
```

**优点**：
- 纯 CSS 实现，无需 JS
- 浏览器原生支持，体验一致
- 自动在右下角显示拉伸手柄

**缺点**：
- 无法自定义拉伸手柄的样式（浏览器自带三角图标）
- 某些浏览器可能样式不太美观

#### 方案二：JS 拖拽实现

通过 JS 监听鼠标事件实现自定义拖拽拉伸，可以自定义手柄样式。

**涉及改动**：
- 在面板右下角添加一个拉伸手柄元素
- 监听 mousedown/mousemove/mouseup 事件动态修改面板宽高

**选用方案**：**方案一（CSS resize）**，简单可靠，样式也够用。

---

## 优化四：一键附带文档内容按钮

### 问题分析

目前只有工具栏的 "📤 问 AI" 按钮可以发送文档内容，但它是自动发送（固定 prompt + 文档内容），用户无法在 AI 对话面板中输入自定义提示词时灵活选择是否附带文档内容。

### 方案设计

在 AI 对话面板的输入区域新增一个 "📎 附带文档" 切换按钮，点击后可将当前文档内容作为上下文发送。

#### 方案细节

1. **UI 位置**：在输入框和发送按钮之间，或输入框上方增加一行操作栏

2. **交互设计**：
   - 按钮为 toggle 状态：点击切换是否附带文档内容
   - 选中状态时显示 `📎 已附带文档`，未选中显示 `📎 附带文档`
   - 发送消息时，如果 toggle 为开启状态，则将文档内容作为一条 `user` 消息插入到 messages 历史中（位于用户当前输入之前）

3. **数据流**（配合优化一的方案）：
   ```
   用户输入 "总结这段内容" + 开启"附带文档"
   ↓
   messages 构造为：
     system: "你是一个文档编辑助手..."
     user:   "以下是我的文档内容：\n\n```\n{文档内容}\n```"
     user:   "总结这段内容"
   ↓
   发送到后端
   ```

4. **状态管理**：
   - toggle 状态存储在 JS 变量中
   - 每次发送消息后保持 toggle 状态不变（用户可以连续发送多轮附带文档的请求）
   - 文档内容从 `editor.value` 实时获取（如果用户切换了文档或修改了内容，发送时使用最新内容）

### 前端改动

在 `static/editor.html` 面板的输入区域增加按钮：

```html
<div class="ai-chat-input-area">
    <button id="aiChatAttachDoc" title="附带当前文档内容" class="attach-btn">📎 附带文档</button>
    <textarea id="aiChatInput" placeholder="输入你的问题..." rows="3"></textarea>
    <button id="aiChatSend">发送</button>
</div>
```

在 `static/ai-chat.js` 中：

```javascript
let attachDocument = false;  // toggle 状态

// 切换附带文档按钮状态
document.getElementById('aiChatAttachDoc')?.addEventListener('click', () => {
    attachDocument = !attachDocument;
    const btn = document.getElementById('aiChatAttachDoc');
    btn.textContent = attachDocument ? '📎 已附带文档' : '📎 附带文档';
    btn.classList.toggle('active', attachDocument);
});
```

发送逻辑中，如果 `attachDocument` 为 true，则从 `editor.value` 读取内容并作为上下文消息插入。

---

## 整体改动汇总

| # | 操作 | 文件 | 改动说明 |
|---|------|------|----------|
| 1 | **修改** | `static/ai-chat.js` | 维护 `messages` 数组；发送时携带完整历史；Markdown 渲染助手消息；新增"附带文档" toggle 逻辑 |
| 2 | **修改** | `internal/handler/ai_handler.go` | `chatRequest` 接收 `messages` 数组替代 `prompt` + `content`；透传给 service |
| 3 | **修改** | `internal/service/ai_service.go` | `ChatStream` 接收 `[]deepseekMsg` 参数，不再内部构造消息 |
| 4 | **修改** | `static/editor.html` | 引入 marked + DOMPurify CDN；输入区域增加"附带文档"按钮 |
| 5 | **修改** | `static/style.css` | AI 面板增加 `resize` 支持 + Markdown 渲染样式 + 附带文档按钮样式 |
| 6 | **不修改** | `cmd/server/main.go` | API 路由不变，无需修改 |

## 交互流程（优化后）

```
初始化:
  messages = [
    {role: "system", content: "你是一个文档编辑助手..."}
  ]

用户输入 "翻译成英文" + 开启"附带文档":
  attachedContent = editor.value  // 实时读取
  messages = [
    {role: "system", ...},
    {role: "user", content: "以下是我的文档内容：\n```\n{attachedContent}\n```"},
    {role: "user", content: "翻译成英文"}
  ]
  POST /api/ai/chat → {messages}

后端收到 messages，透传给 DeepSeek API
  ↓
流式返回 AI 响应
  ↓
前端逐 chunk 更新显示 (Markdown 渲染)
  ↓
完整响应后，push {role: "assistant", content: fullContent} 到 messages
  ↓
用户可连续对话 (下一轮携带全部历史)
```

## 注意事项

1. **消息历史长度**：随着对话进行，`messages` 数组会不断增长，可能导致 token 超限。建议在前端对 messages 长度做限制（如保留最近 20 轮对话），或显示 token 计数。

2. **Markdown XSS**：使用 `marked` + `DOMPurify` 双重防护，禁止用户输入或 AI 返回内容中嵌入恶意脚本。

3. **文档内容实时性**："附带文档"功能发送时从 `editor.value` 实时读取，确保发送的是最新内容。

4. **兼容性**：CSS `resize` 在主流浏览器中均支持（IE 不支持），本项目目标为现代浏览器，无兼容性问题。

5. **CDN 加载**：marked 和 DOMPurify 通过 CDN 加载，需要网络联通。如遇离线场景可考虑将库下载到 `/static/lib/` 目录本地引用。