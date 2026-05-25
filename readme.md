# 本地文档编辑器 (go-doc-editor)

一个基于 Go 构建的轻量级本地文档编辑器，支持用户认证、文档 CRUD 操作和 AI 对话助手。

---

## 项目架构概览

本项目采用 **Go 标准项目布局**，按职责分层设计：

1. **配置层** (`config`)：管理服务端口、文档存储路径等运行参数。
2. **模型层** (`model`)：定义前后端交互的数据结构（请求/响应格式）。
3. **业务逻辑层** (`service`)：实现文件读写、路径安全校验、目录操作、AI API 调用等核心功能。
4. **HTTP 处理层** (`handler`)：解析 HTTP 请求，调用业务层，封装 JSON 或 SSE 响应。
5. **认证层** (`auth` / `user`)：JWT 令牌生成验证、用户注册登录与密码加密存储。
6. **中间件层** (`middleware`)：HTTP 中间件，如 JWT 认证拦截。
7. **入口层** (`cmd/server`)：组装各模块、注册路由、启动服务。
8. **静态资源** (`static`)：前端 HTML/JS/CSS，提供欢迎页、编辑器界面和 AI 对话面板。

该架构实现了**关注点分离**，便于单独测试、扩展和维护。

---

## 目录结构与文件职责

```
go-doc-editor/
├── cmd/
│   └── server/
│       └── main.go                     # 程序入口：加载配置、初始化各层、注册路由、启动 HTTP 服务
├── internal/
│   ├── config/
│   │   └── config.go                   # 配置管理：定义 Config 结构体，支持命令行参数和环境变量读取
│   ├── model/
│   │   └── file.go                     # 数据模型：定义统一响应结构 (Response) 和保存请求结构 (SaveFileRequest)
│   ├── auth/
│   │   └── jwt.go                      # JWT 工具：生成和验证令牌，存储用户名声明
│   ├── user/
│   │   └── store.go                    # 用户存储：基于 JSON 文件的用户管理，bcrypt 密码加密
│   ├── service/
│   │   ├── file_service.go             # 文件业务逻辑：用户隔离的 CRUD 操作、路径安全校验（防目录遍历）
│   │   └── ai_service.go               # AI 服务：调用 DeepSeek API 流式接口，通过回调逐块返回
│   ├── handler/
│   │   ├── file_handler.go             # 文件 HTTP 处理器：为文件 API 端点提供处理函数，从上下文获取用户名
│   │   ├── auth_handler.go             # 认证 HTTP 处理器：注册/登录 API，返回 JWT 令牌
│   │   └── ai_handler.go               # AI HTTP 处理器：接收对话消息，SSE 流式转发 AI 响应
│   └── middleware/
│       └── auth.go                     # 认证中间件：验证 Bearer Token，将用户名注入请求上下文
├── static/
│   ├── index.html                      # 欢迎页：根据登录状态显示"登录/注册"或"进入编辑器"按钮
│   ├── login.html                      # 登录页面：用户名/密码表单，成功后存储 token 跳转
│   ├── register.html                   # 注册页面：用户名/密码表单，注册成功后跳转登录
│   ├── editor.html                     # 文档编辑器：文档列表、编辑器、新建/保存/删除 + AI 对话面板
│   ├── editor.js                       # 编辑器逻辑：自动携带 token 的 fetch、文件操作、状态管理
│   ├── ai-chat.js                      # AI 对话模块：SSE 接收、对话历史维护、Markdown 渲染、附带文档
│   ├── auth.js                         # 通用 Token 管理：存储/获取/移除、登录状态判断、跳转
│   └── style.css                       # 全局样式
├── go.mod                              # Go 模块定义：声明模块路径和依赖
└── go.sum                              # 依赖校验文件（自动生成）
```

---

## 各文件核心功能速览

| 文件路径 | 核心职责 | 关键函数/类型 |
|---------|----------|--------------|
| `cmd/server/main.go` | 服务启动与装配 | `main()` – 加载配置 → 创建各层 → 注册公开/受保护路由 → 启动监听 |
| `internal/config/config.go` | 配置加载 | `Load()` – 解析 `-port` 和 `-storage` 命令行参数，支持环境变量覆盖 |
| `internal/model/file.go` | 数据契约 | `Response` – 统一 API 返回格式；`SaveFileRequest` – 保存文件时的 JSON 体 |
| `internal/auth/jwt.go` | JWT 认证 | `GenerateToken` – 生成 24 小时有效令牌；`ValidateToken` – 验证并解析声明 |
| `internal/user/store.go` | 用户管理 | `CreateUser` – 注册（bcrypt 加密）；`ValidateUser` – 登录验证 |
| `internal/service/file_service.go` | 文件操作 + 安全校验 | `ListFiles`, `ReadFile`, `SaveFile`, `DeleteFile`；内置 `safeFilePath` 防止路径遍历 |
| `internal/service/ai_service.go` | AI 流式对话 | `ChatStream` – 调用 DeepSeek 流式 API，逐块回调返回；`Chat` – 非流式封装 |
| `internal/handler/file_handler.go` | 文件 HTTP 适配 | `ListFiles`, `ReadFile`, `SaveFile`, `DeleteFile`；从 JWT 上下文提取用户名 |
| `internal/handler/auth_handler.go` | 认证 HTTP 适配 | `Register` – 注册新用户；`Login` – 验证身份并返回 token |
| `internal/handler/ai_handler.go` | AI HTTP 适配 | `Chat` – 接收消息列表，以 SSE 格式流式返回 AI 响应 |
| `internal/middleware/auth.go` | 认证拦截 | `AuthMiddleware` – 验证 `Authorization: Bearer <token>` 头，注入 username 到 context |
| `static/index.html` | 欢迎主页 | 根据登录状态显示登录/注册入口或跳转编辑器 |
| `static/editor.html` | 编辑器主页面 | 完整文档编辑器 + AI 对话面板 |
| `static/editor.js` | 编辑器逻辑 | 自动拦截 fetch 添加 token，401 自动跳转登录 |
| `static/ai-chat.js` | AI 对话交互 | SSE 接收、消息历史管理、Markdown 渲染、附带文档切换 |
| `static/auth.js` | Token 工具 | `getToken`, `setToken`, `removeToken`, `isLoggedIn`, `redirectToLogin` |

---

## 分层数据流

### 文档操作流程

```
浏览器 (editor.html / editor.js)
    │
    │ HTTP 请求 (Authorization: Bearer <token>)
    ▼
main.go (路由分发)
    │
    ├── /api/register, /api/login → auth_handler.go (无需认证)
    │
    └── /api/files, /api/file/*, /api/ai/chat
        │
        ▼
    middleware/auth.go (验证 JWT Token，提取用户名注入 context)
        │
        ▼
    handler/*.go (解析请求、参数校验、获取用户名)
        │
        ▼
    service/*.go (执行业务逻辑)
        │
        ▼
    本地文件系统 / DeepSeek API
        │
        ▼  (返回结果)
    service → handler → JSON/SSE → 浏览器
```

### AI 对话流程

```
用户编辑文档 → 点击工具栏 "📤 问 AI" 或在 AI 面板输入
    │
    ▼
前端维护 messages 数组（含 system/user/assistant 历史）
    │
    │ POST /api/ai/chat (JSON: {messages})
    │ Authorization: Bearer <token>
    ▼
后端 middleware/auth.go (JWT 验证)
    │
    ▼
handler/ai_handler.go (解析 messages，设置 SSE 响应头)
    │
    ▼
service/ai_service.go → POST https://api.deepseek.com/v1/chat/completions (stream)
    │
    ▼  (SSE 逐块转发)
handler → text/event-stream → 前端 ReadableStream
    │
    ▼
ai-chat.js (实时更新对话气泡，Markdown 渲染)
    │
    ▼
完整响应后追加 {role: "assistant"} 到 messages 历史
```

---

## 核心安全设计

1. **JWT 认证**：使用 `golang-jwt/jwt/v5` 生成 HS256 签名令牌，24 小时有效期。所有文档和 AI 接口均需携带有效 token。
2. **密码加密**：使用 `bcrypt` 对用户密码进行哈希存储，不保存明文。
3. **路径遍历防护**：`service/file_service.go` 中 `safeFilePath` 检查文件名是否包含 `..`、`/`、`\`，并确保最终绝对路径位于用户专属存储目录内。
4. **用户隔离**：每个用户拥有独立文档目录（`{storageDir}/{username}`），无法访问其他用户文件。
5. **双重验证**：前端和后端均进行文件名和登录状态校验。

---

## API 接口文档

### 公开接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/register` | 注册新用户（username, password） |
| POST | `/api/login` | 登录，返回 JWT token |

### 受保护接口（需要 `Authorization: Bearer <token>`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/files` | 获取当前用户的文件列表 |
| GET | `/api/file/{filename}` | 读取指定文件内容 |
| POST | `/api/file/{filename}` | 保存/覆盖指定文件 |
| DELETE | `/api/file/{filename}` | 删除指定文件 |
| POST | `/api/ai/chat` | AI 对话（SSE 流式返回） |

### AI 对话请求格式

```json
{
  "messages": [
    {"role": "system", "content": "你是一个文档编辑助手..."},
    {"role": "user", "content": "你好，帮我翻译这段文字"},
    {"role": "assistant", "content": "好的，请提供需要翻译的内容"}
  ]
}
```

### AI 对话响应格式（SSE）

```
data: {"type":"start"}
data: {"type":"chunk","delta":"这是"}
data: {"type":"chunk","delta":"回复内容"}
data: {"type":"done","content":"这是回复内容"}
```

---

## 前端特性

- **欢迎页** (`index.html`)：根据登录状态动态显示登录/注册按钮或编辑器入口
- **用户认证**：独立的登录/注册页面，成功后将 token 存储于 `localStorage`
- **文档编辑器** (`editor.html`)：文件列表、文本编辑、新建/保存/删除操作
- **自动认证**：`editor.js` 重写 `window.fetch` 自动为所有请求添加 `Authorization` 头，401 响应自动跳转登录
- **AI 对话面板**：
  - 浮动按钮呼出/隐藏
  - 完整的对话历史管理（前端维护 messages 数组）
  - Markdown 渲染（使用 marked + DOMPurify 防 XSS）
  - 可拉伸面板（CSS `resize`）
  - "📎 附带文档"开关，灵活选择是否将当前文档作为上下文发送
  - 工具栏"📤 问 AI"一键发送当前文档

---

## 运行与测试

1. 安装依赖：
   ```bash
   go mod tidy
   ```

2. 编译运行：
   ```bash
   go run cmd/server/main.go
   ```
   或指定端口和存储目录：
   ```bash
   go run cmd/server/main.go -port :8080 -storage "D:\my_docs"
   ```

3. 浏览器访问 `http://localhost:3000`。

4. 注册账号 → 登录 → 进入文档编辑器 → 开始编辑文档。

---

## 扩展接口说明

该项目设计为易于扩展：

- **增加文件操作**（如重命名、移动）：在 `service/file_service.go` 中添加方法，在 `handler/file_handler.go` 中增加对应路由处理，在 `main.go` 注册路由。
- **更换 AI 模型**：修改 `service/ai_service.go` 中的 `Model` 字段和 API URL 即可。
- **数据库存储**：替换 `user/store.go` 的 JSON 文件存储为数据库实现，接口保持不变。
- **添加中间件**（日志、CORS、速率限制）：在 `main.go` 的路由注册处包装 handler 函数。
- **AI 增强**：修改 `ai-chat.js` 的 system prompt 或消息处理逻辑即可调整 AI 行为。

---

## 依赖

- [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) — JWT 令牌生成与验证
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) — bcrypt 密码加密
- [marked](https://marked.js.org/) — Markdown 渲染（前端 CDN）
- [DOMPurify](https://github.com/cure53/DOMPurify) — XSS 防护（前端 CDN）