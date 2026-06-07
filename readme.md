# 本地文档编辑器 (go-doc-editor)

一个基于 Go 构建的轻量级**知识收纳整理**工具，支持文档管理、三级分类体系、关键语句标注与跨文档关联、AI 对话助手。

> **📚 开发规范体系**：本项目已建立完整的开发流程规范，涵盖
> 分支策略、提交规范、代码风格、架构规范、测试要求、范式 Super 等。
> 从 [docs/INDEX.md](./docs/INDEX.md) 开始阅读。
> 项目历史演进参见 [docs/00-项目历史沿革.md](./docs/00-项目历史沿革.md)。

---

## 项目架构概览

本项目采用 **Go 标准项目布局** + **MySQL 数据库**，按职责分层设计：

```
前端 (static/) → Handler (HTTP 适配) → Service (业务逻辑) → 外部资源
                                       ├── MySQL（用户/文档/分类/标注）
                                       └── DeepSeek API（AI 对话）
```

1. **数据库层** (`db`)：MySQL 连接管理，自动建表（users / login_logs / documents / categories / annotations / ai_keys）
2. **配置层** (`config`)：支持 config.json / 命令行参数 / 环境变量三种配置方式
3. **模型层** (`model`)：定义前后端交互的数据结构（请求/响应格式）
4. **存储层** (`docstore`)：基于 MySQL 的文档、分类、标注 CRUD 操作
5. **业务逻辑层** (`service`)：文档 CRUD、分类管理、标注管理等核心业务
6. **HTTP 处理层** (`handler`)：解析 HTTP 请求，调用业务层，封装 JSON 或 SSE 响应
7. **认证层** (`auth` / `user`)：JWT 令牌生成验证、用户注册登录、密码加密存储
8. **中间件层** (`middleware`)：HTTP 中间件，如 JWT 认证拦截
9. **入口层** (`cmd/server`)：组装各模块、注册路由、启动服务
10. **静态资源** (`static`)：前端 HTML/JS/CSS

该架构实现了**关注点分离**，便于单独测试、扩展和维护。

---

## 快速开始

### 前置要求

- Go 1.25+
- MySQL 8.0+
- DeepSeek API Key（或其他兼容的 OpenAI 格式 API Key）

> **💡 AI-Key 不用在这里配置！** 启动后登录 → 点击右上角 👤 → 在个人主页添加自己的 AI-Key。

### 1. 配置 config.json

`config.json` 已包含数据库等默认配置：

```json
{
    "port": ":3000",
    "storage": "C:\\doxreader",
    "mysql_dsn": "root:密码@tcp(localhost:3306)/godoxedit?charset=utf8mb4&parseTime=true"
}
```

先创建 MySQL 数据库（仅首次）：

```bash
mysql -u root -p
CREATE DATABASE godoxedit CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
exit
```

### 2. 运行

```bash
# 一键启动，无需任何参数！
go run cmd/server/main.go
```

### 3. 访问

浏览器打开 `http://localhost:3000` → 注册账号 → 登录 → 开始使用。

### 配置优先级

```
环境变量（最高）> 命令行参数 > config.json（最低）
```

支持的环境变量：`PORT`、`MYSQL_DSN`

---

## 目录结构与文件职责

```
go-doc-editor/
├── cmd/
│   └── server/
│       └── main.go                        # 程序入口
│
├── internal/
│   ├── db/
│   │   └── mysql.go                       # MySQL 连接管理 & 自动建表
│   ├── config/
│   │   └── config.go                      # 配置管理（config.json + flag + env）
│   ├── model/
│   │   └── file.go                        # 数据模型（Response / SaveFileRequest）
│   ├── auth/
│   │   └── jwt.go                         # JWT 生成与验证
│   ├── user/
│   │   └── store.go                       # MySQL 版用户存储（注册/登录/日志查询）
│   ├── docstore/
│   │   └── doc_store.go                   # 文档数据库存储 + 分类 CRUD + 标注 CRUD
│   ├── service/
│   │   ├── file_service.go                # 文档/分类/标注业务逻辑
│   │   └── ai_service.go                  # DeepSeek API 调用（流式/非流式）
│   ├── handler/
│   │   ├── auth_handler.go                # 注册/登录 API
│   │   ├── file_handler.go                # 文档操作 API（含分类筛选）
│   │   ├── category_handler.go            # 分类树 CRUD + 文档归属分类
│   │   ├── annotation_handler.go          # 关键语句标注 + 跨文档引用
│   │   ├── ai_handler.go                  # AI 对话 API（SSE 流式）
│   │   └── profile_handler.go             # 用户主页 API
│   └── middleware/
│       └── auth.go                        # JWT 认证拦截中间件
│
├── static/
│   ├── index.html                         # 欢迎主页（登录/注册入口）
│   ├── login.html                         # 登录页面
│   ├── register.html                      # 注册页面
│   ├── editor.html                        # 文档编辑器 + AI 对话面板
│   ├── editor.js                          # 编辑器逻辑（fetch 拦截、401 处理）
│   ├── ai-chat.js                         # AI 对话模块（SSE、Markdown、角色扮演）
│   ├── auth.js                            # Token 管理工具
│   ├── profile.html                       # 🆕 用户主页
│   ├── profile.js                         # 🆕 用户主页逻辑
│   └── style.css                          # 全局样式
│
├── docs/                                  # 开发规范与项目文档
│   ├── INDEX.md                           # 规范索引入口
│   ├── 01-总体原则.md ~ 12-功能开发流程规范.md
│   ├── adr/                               # 架构决策记录
│   ├── specs/                             # 功能规格文档
│   ├── images/                            # 流程截屏
│   └── logs/功能日志/                       # 开发日志
│
├── archive/                               # 📦 历史文档归档
│   ├── 添加新功能.md
│   ├── AI集成方案.md
│   ├── AI对话优化方案.md
│   └── DEVELOPMENT_WORKFLOW.md
│
├── config.json                            # 配置文件（AI Key 用占位符）
├── go.mod / go.sum
└── readme.md
```

---

## API 接口文档

### 公开接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/register` | 注册新用户（username, password） |
| POST | `/api/login` | 登录，返回 JWT token |
| GET | `/api/ai/roles` | 获取 AI 角色扮演列表 |

### 受保护接口（需 `Authorization: Bearer <token>`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/files` | 获取当前用户的文件列表 |
| GET | `/api/file/{filename}` | 读取指定文件内容 |
| POST | `/api/file/{filename}` | 保存/覆盖指定文件 |
| DELETE | `/api/file/{filename}` | 删除指定文件 |
| POST | `/api/ai/chat` | AI 对话（SSE 流式返回，使用用户配置的 Key） |
| GET | `/api/user/profile` | 获取用户信息、登录历史、AI Key 列表 |
| GET | `/api/user/ai-keys` | 获取 AI Key 列表 |
| POST | `/api/user/ai-keys` | 新增 AI Key |
| PUT | `/api/user/ai-keys/{id}/activate` | 激活指定 Key |
| DELETE | `/api/user/ai-keys/{id}` | 删除指定 Key |
| GET | `/api/categories` | 获取分类树（含子分类） |
| POST | `/api/categories` | 创建分类（name / parentId） |
| PUT | `/api/categories/{id}` | 重命名分类 |
| DELETE | `/api/categories/{id}` | 删除分类（有子分类或文档时拒绝） |
| PUT | `/api/file/{filename}/category` | 设置文档所属分类 |
| GET | `/api/file/{filename}/category` | 获取文档当前分类 |
| GET | `/api/files?category={id}` | 按分类筛选文档列表 |
| GET | `/api/files?uncategorized=1` | 获取未分类文档列表 |
| POST | `/api/annotations` | 创建标注（选中文本 + 关联文档 + 评语） |
| GET | `/api/file/{filename}/annotations` | 获取文档的所有标注 |
| GET | `/api/file/{filename}/references` | 获取引用某文档的标注（被引用于） |
| DELETE | `/api/annotations/{id}` | 删除标注 |

### AI 对话请求格式

```json
{
  "messages": [
    {"role": "system", "content": "你是一个文档编辑助手..."},
    {"role": "user", "content": "你好，帮我翻译这段文字"}
  ]
}
```

### AI 对话响应格式（SSE）

```
data: {"type":"start"}
data: {"type":"chunk","delta":"回复片段"}
data: {"type":"done","content":"完整回复"}
```

---

## 核心安全设计

1. **JWT 认证**：HS256 签名令牌，24 小时有效期
2. **密码加密**：bcrypt 哈希存储，不保存明文
3. **SQL 注入防护**：全部使用参数化查询（`?` 占位符）
4. **用户隔离**：所有查询按 username 过滤，数据库天然防止跨用户访问
5. **分类深度校验**：应用层限制最多三级分类（大类→中类→小类）
6. **同级重名保护**：创建分类时检查同级下是否已存在同名分类
7. **删除保护**：有子分类或有文档归属的分类不可删除
8. **API 密钥安全**：AI Key 由用户在个人主页自行配置，无需命令行参数

---

## 前端特性

- **欢迎主页**：根据登录状态动态切换登录/编辑器入口
- **用户认证**：登录/注册页面，Token 存储于 localStorage
- **文档编辑器**：文件列表 + 文本编辑 + 新建/保存/删除
- **自动认证**：fetch 自动携带 Token，401 自动跳转登录
- **📂 三级分类体系**：
  - 侧边栏树形分类浏览，点击折叠/展开
  - 新建/重命名/删除分类（大类→中类→小类，限三级）
  - 文档可归属到任意一级分类
  - 按分类筛选文档列表
  - 未分类文档独立入口
- **📌 关键语句标注**：
  - 选中文本弹出标注按钮
  - 可关联到其他文档 + 添加评语
  - 侧边栏标注面板展示当前文档的所有标注
  - 被引用于提示（反向引用）
  - 工具栏标注按钮备用入口
- **🔄 全局最小化**：
  - 整个编辑器可一键收起/展开（平滑缩放过渡）
  - 收起后 ⚫ 和 💬 按钮可自由拖拽
  - 展开时按钮丝滑飞回原位
- **AI 对话面板**：
  - 浮动按钮呼出/隐藏
  - 完整对话历史管理（前端维护 messages 数组）
  - Markdown 渲染（marked + DOMPurify 防 XSS）
  - **角色扮演**：5 种预设角色，可切换/关闭
  - **附带文档**：将当前文档作为上下文发送
  - 面板可拖拽、边框可拉伸
- **设计风格**：极简黑白主题，无 emoji 图标（纯 CSS 视觉指示器）
- **用户主页**：右上角白底圆形按钮，展示用户名、注册时间、登录历史
- **AI-Key 管理**：在 profile 页面自行添加/切换/删除 API-Key，支持 DeepSeek-v4-Flash 和 DeepSeek-v4-Pro 模型

---

## 数据库

首次启动自动建表，无需手动执行 SQL：

```sql
-- 用户表
CREATE TABLE users (
    username      VARCHAR(255) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 登录日志表
CREATE TABLE login_logs (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    username   VARCHAR(255) NOT NULL,
    login_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX (username),
    INDEX (login_time)
);

-- AI-Key 配置表（用户可添加多个 Key，切换激活）
CREATE TABLE ai_keys (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    username   VARCHAR(255) NOT NULL,
    key_name   VARCHAR(100) NOT NULL,
    api_key    VARCHAR(255) NOT NULL,
    api_url    VARCHAR(255) NOT NULL DEFAULT 'https://api.deepseek.com/chat/completions',
    model      VARCHAR(100) NOT NULL DEFAULT 'deepseek-v4-flash',
    is_active  TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX (username)
);

-- 文档存储表（文档内容以 LONGTEXT 存储在数据库中）
CREATE TABLE documents (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    username   VARCHAR(255) NOT NULL,
    filename   VARCHAR(255) NOT NULL,
    content    LONGTEXT,
    category_id BIGINT DEFAULT NULL COMMENT '所属分类ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE INDEX (username, filename),
    INDEX (username),
    INDEX idx_doc_category (username, category_id)
);

-- 三级分类树（parent_id 自引用，应用层限制三级）
CREATE TABLE categories (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    username   VARCHAR(255) NOT NULL,
    name       VARCHAR(100) NOT NULL,
    parent_id  BIGINT DEFAULT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX (username),
    INDEX (username, parent_id),
    UNIQUE INDEX (username, parent_id, name)
);

-- 关键语句标注表（支持跨文档关联）
CREATE TABLE annotations (
    id               BIGINT AUTO_INCREMENT PRIMARY KEY,
    username         VARCHAR(255) NOT NULL,
    source_filename  VARCHAR(255) NOT NULL,
    selected_text    TEXT NOT NULL,
    target_filename  VARCHAR(255) DEFAULT NULL,
    comment          TEXT,
    position_start   INT NOT NULL DEFAULT 0,
    position_end     INT NOT NULL DEFAULT 0,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX (username, source_filename),
    INDEX (username, target_filename)
);
```

---

## 依赖

| 依赖 | 用途 |
|------|------|
| [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | MySQL 数据库驱动 |
| [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) | JWT 令牌 |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | bcrypt 密码加密 |
| [marked](https://marked.js.org/) | Markdown 渲染（前端 CDN） |
| [DOMPurify](https://github.com/cure53/DOMPurify) | XSS 防护（前端 CDN） |

---

## 扩展指南

- **增加文件操作**：`service/file_service.go` 加方法 → `handler/file_handler.go` 加处理 → `main.go` 加路由
- **更换 AI 模型**：修改 `service/ai_service.go` 的 `Model` 和 API URL
- **新增数据库表**：在 `db/mysql.go` 的 `InitTables()` 中添加 SQL
- **添加中间件**：在 `main.go` 的路由处包装 handler

---

## 开发流程范式 · 建立历程

> 本项目的开发流程规范是在**真实的开发对话中逐步演进、迭代优化**而成的。

### 范式建立历程

```
[初期] 需求 → 直接开发 → 测试 → 修改（无固定流程）
   │
   ▼
[第一次迭代] 要求记录：功能需求 / 测试反馈 / 最终评价
   │
   ▼
[第二次迭代] 调研业界标准 → 定制为 8 步范式 A/B/C 流程
   │
   ▼
[第三次迭代] 通过三个真实功能验证：
   ├── AI 角色扮演         ✓ 1 次迭代
   ├── AI 面板交互优化     ✓ 3 次迭代
   └── 用户主页 + MySQL    ✓ 2 次迭代
   │
   ▼
[最终] docs/01~12 完整规范体系
```

| 截图 | 阶段 | 内容 |
|:----:|------|------|
| ![范式提出](docs/images/屏幕截图%202026-06-06%20161109.png) | 范式雏形 | 要求调研流程范式，提出 3 种类型 |
| ![规范设计](docs/images/屏幕截图%202026-06-06%20161126.png) | 规范设计 | 讨论各步骤产出物和 Log 规范 |
| ![试行验证](docs/images/屏幕截图%202026-06-06%20161135.png) | 首次试行 | AI 角色扮演验证范式 A |
| ![流程沉淀](docs/images/屏幕截图%202026-06-06%20161204.png) | 流程固化 | 正式纳入项目规范体系 |

详细规范见 [`docs/12-功能开发流程规范.md`](docs/12-功能开发流程规范.md)。

---

## 更新历史

| 日期 | 版本 | 变更 |
|------|------|------|
| 2026-06-07 | v0.9.0 | 🎨 前端 UI 全面优化：设计系统重构、分类树折叠、全局最小化/拖拽、去 emoji 图标、极简主题 |
| 2026-06-07 | v0.8.0 | 🧠 知识收纳整理系统：三级分类体系（大类→中类→小类）+ 关键语句标注与跨文档关联 + 范式 Super 流程 |
| 2026-06-06 | v0.7.0 | 🗄️ 文档存储迁移至数据库：新增 `documents` 表，`docstore` 包，移除文件系统依赖，新增 ADR-005 |
| 2026-06-06 | v0.6.0 | 🎯 AI-Key 管理系统：用户自主配置 Key/URL/模型；移除 `--ai-key` 参数；模型升级至 deepseek-v4-flash/v4-pro |
| 2026-06-06 | v0.5.0 | 🔑 AI-Key 管理功能：新增 `ai_keys` 表，用户主页可添加/激活/删除 Key |
| 2026-06-06 | v0.4.0 | 新增配置文件 `config.json`，支持三种配置优先级 |
| 2026-06-06 | v0.3.0 | 新增用户主页 + MySQL 数据库（users + login_logs 表） |
| 2026-06-06 | v0.2.0 | AI 面板交互优化（拖拽 + 边框拉伸 + 角色选择器美化） |
| 2026-06-06 | v0.1.0 | 新增 AI 角色扮演功能（5 种预设角色） |
| 2026-05-20 | v0.0.4 | AI 对话优化（历史上下文、Markdown 渲染、附带文档） |
| 2026-05-15 | v0.0.3 | AI 集成（DeepSeek 后端代理 + SSE 流式响应） |
| 2026-05-01 | v0.0.2 | 用户认证（JWT + bcrypt + 用户隔离） |
| 2026-04-xx | v0.0.1 | 项目初始化，基础文档 CRUD |
