# 任务拆解：用户主页 + MySQL 数据库

- **日期**: 2026-06-06

---

## 依赖图谱

```
Task 1: 添加 MySQL 驱动依赖
   │
   ├── Task 2: 创建 db/mysql.go 连接管理 + 建表
   │      │
   │      └── Task 3: 重写 user/store.go 为 MySQL
   │             │
   │             ├── Task 4: 更新 config 增加 MySQLDSN
   │             │
   │             ├── Task 5: 修改 auth_handler（记录登录日志）
   │             │
   │             ├── Task 6: 新建 profile_handler
   │             │
   │             └── Task 7: 更新 main.go 组装
   │                    │
   │                    ├── Task 8: 创建 profile.html + profile.js
   │                    │
   │                    └── Task 9: 添加用户按钮（index + editor + CSS）
```

## Task 列表

### Phase 1：数据库基础

---

**Task 1：添加 MySQL 驱动依赖**

`go get github.com/go-sql-driver/mysql` → 更新 go.mod / go.sum

**涉及文件**：`go.mod`、`go.sum`

---

**Task 2：创建 internal/db/mysql.go**

```go
func NewDB(dsn string) (*sql.DB, error)     // 连接 MySQL
func InitTables(db *sql.DB) error            // 初始化 users + login_logs 表
```

**涉及文件**：`internal/db/mysql.go`（新）

---

### Phase 2：后端重构

---

**Task 3：重写 internal/user/store.go**

保留 `CreateUser`、`ValidateUser` 签名不变，新增方法：
- `GetUserInfo(username) (createdAt, error)`
- `RecordLogin(username) error`
- `GetLoginLogs(username, limit) ([]LoginLog, error)`

构造函数改为 `NewStore(db *sql.DB)`。

**涉及文件**：`internal/user/store.go`

---

**Task 4：更新 internal/config/config.go**

Config 结构体新增 `MySQLDSN string` 字段，
命令行参数新增 `-mysql-dsn`。

**涉及文件**：`internal/config/config.go`

---

**Task 5：修改 internal/handler/auth_handler.go**

Login 成功后调用 `userStore.RecordLogin(username)`。
Register 中注册时间由 MySQL 自动记录。

**涉及文件**：`internal/handler/auth_handler.go`

---

**Task 6：新建 internal/handler/profile_handler.go**

新增 `GET /api/user/profile` 端点，
返回：用户名、注册时间、最近登录时间、登录列表。

**涉及文件**：`internal/handler/profile_handler.go`（新）

---

**Task 7：更新 cmd/server/main.go**

- 新增 `-mysql-dsn` 参数
- 启动时连接 MySQL + InitTables
- 创建 MySQL 版 userStore
- 注册 `/api/user/profile` 路由

**涉及文件**：`cmd/server/main.go`

---

### Phase 3：前端

---

**Task 8：创建用户主页**

`static/profile.html` + `static/profile.js`：
- 自动从 JWT 获取 token
- 请求 `/api/user/profile` 展示信息
- 简洁卡片式布局

**涉及文件**：`static/profile.html`（新）、`static/profile.js`（新）

---

**Task 9：右上角用户按钮**

在 `index.html`（登录后状态）和 `editor.html` 右上角添加圆形按钮。
点击跳转 `profile.html`。

添加 CSS 样式（圆形按钮 + 悬停效果）。

**涉及文件**：`static/index.html`、`static/editor.html`、`static/style.css`
