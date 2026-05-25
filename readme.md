以下为您整理的项目架构与各文件职责概述，聚焦于**分层设计**和**模块功能**，不包含完整代码实现。

---

## 项目架构概览

本项目采用 **Go 标准项目布局**，将代码按职责分为四层：

1. **配置层** (`config`)：管理服务端口、文档存储路径等运行参数。
2. **模型层** (`model`)：定义前后端交互的数据结构（请求/响应格式）。
3. **业务逻辑层** (`service`)：实现文件读写、路径安全校验、目录操作等核心功能。
4. **HTTP 处理层** (`handler`)：解析 HTTP 请求，调用业务层，封装 JSON 响应。
5. **入口层** (`cmd/server`)：组装各模块、注册路由、启动服务。
6. **静态资源** (`static`)：前端 HTML/JS/CSS，提供编辑界面。

该架构实现了**关注点分离**，便于单独测试、扩展和维护。

---

## 目录结构与文件职责

```
go-doc-editor/
├── cmd/
│   └── server/
│       └── main.go                # 程序入口：加载配置、初始化各层、注册路由、启动 HTTP 服务
├── internal/                      # 内部包（不对外暴露）
│   ├── config/
│   │   └── config.go              # 配置管理：定义 Config 结构体，支持命令行参数和环境变量读取
│   ├── model/
│   │   └── file.go                # 数据模型：定义统一响应结构 (Response) 和保存请求结构 (SaveFileRequest)
│   ├── service/
│   │   └── file_service.go        # 业务逻辑层：实现文件列表、读、写、删除，及路径安全校验（防目录遍历）
│   └── handler/
│       └── file_handler.go        # HTTP 处理层：为每个 API 端点提供处理函数，解析请求参数、调用 service、返回 JSON
├── static/
│   └── index.html                 # 前端单页面：文档列表展示、编辑器、新建/保存/删除操作，通过 Fetch API 与后端通信
├── go.mod                         # Go 模块定义：声明模块路径和依赖
└── go.sum                         # 依赖校验文件（自动生成）
```

---

## 各文件核心功能速览

| 文件路径 | 核心职责 | 关键函数/类型 |
|---------|----------|--------------|
| `cmd/server/main.go` | 服务启动与装配 | `main()` – 加载配置 → 创建 service → 创建 handler → 注册路由 → 启动监听 |
| `internal/config/config.go` | 配置加载 | `Load()` – 解析 `-port` 和 `-storage` 命令行参数，支持环境变量覆盖 |
| `internal/model/file.go` | 数据契约 | `Response` – 统一 API 返回格式；`SaveFileRequest` – 保存文件时的 JSON 体 |
| `internal/service/file_service.go` | 文件操作 + 安全校验 | `ListFiles`, `ReadFile`, `SaveFile`, `DeleteFile`；内部 `safeFilePath` 防止路径遍历 |
| `internal/handler/file_handler.go` | HTTP 请求适配 | `ListFiles`, `ReadFile`, `SaveFile`, `DeleteFile` – 分别对应四个 API 端点；辅助函数 `writeJSON`/`writeError` |
| `static/index.html` | 前端交互界面 | 使用原生 JS 和 Fetch API 实现文档列表、编辑器、保存、删除、新建等完整 CRUD 操作 |

---

## 分层数据流

```
浏览器 (index.html)
    │
    │ HTTP 请求 (GET /api/files, POST /api/file/xxx, ...)
    ▼
main.go (路由分发)
    │
    ▼
handler/file_handler.go (解析请求、参数校验)
    │
    ▼
service/file_service.go (执行业务逻辑：读/写/删文件，路径安全检测)
    │
    ▼
本地文件系统 (C:\doxreader)
    │
    ▼  (返回结果)
service → handler → JSON 响应 → 浏览器
```

---

## 核心安全设计

- **路径遍历防护**：`service/file_service.go` 中的 `safeFilePath` 函数会检查文件名是否包含 `..` 或 `/`、`\`，并确保最终绝对路径仍位于指定存储目录内。
- **文件名校验**：前端也做同名检查，后端双重验证。

---

## 扩展接口说明

该项目设计为易于扩展：

- 若需增加**文件重命名**功能：在 `service` 中添加 `RenameFile` 方法，在 `handler` 中增加对应路由处理。
- 若需更换存储后端（如数据库）：只需修改 `service` 层实现，`handler` 和 `main` 无需变动。
- 若需添加**中间件**（日志、CORS）：可在 `main.go` 的路由注册处包装 `handler` 函数。

---

此架构清晰展示了 Go Web 开发的标准分层模式，适合作为教学示例和后续项目的起点。