# 技术方案设计：用户主页 + MySQL 数据库

- **日期**: 2026-06-06

---

## 方案对比：数据库迁移策略

### 方案 A：重写 user.Store 为 MySQL 实现（选中 ✅）

保持 `user.Store` 结构体不变（`CreateUser`、`ValidateUser`），
内部实现从 JSON 文件替换为 MySQL 查询。

**优点**：
- handler 层零改动
- 路由注册零改动
- 改动集中在 `internal/user/` 包内
- 最快的实施路径

**缺点**：
- 旧的 JSON 用户数据需要迁移（可提供独立迁移脚本）

### 方案 B：抽象 UserRepository 接口 + MySQL 实现
- 先定义接口，再分别实现 JSON 版和 MySQL 版
- → ❌ 当前项目规模不需要双层抽象，过度设计

### 方案 C：保留 JSON + 增加 MySQL 双写
- 双写过渡复杂，数据一致性问题多
- → ❌

**选中方案 A**。

---

## 方案对比：数据库配置方式

### 方案 A：命令行参数 `-mysql-dsn`（选中 ✅）

```bash
go run cmd/server/main.go --ai-key=xxx -mysql-dsn "root:13579@tcp(localhost:3306)/godoxedit?charset=utf8mb4&parseTime=true"
```

**优点**：密码不硬编码、符合现有 `--ai-key` 模式、.gitignore 自动保护

### 方案 B：环境变量
- 可行，但项目当前已使用命令行参数模式，保持一致更好

---

## 详细架构

### 数据流

```
注册流程（修改 Register handler）：
POST /api/register {username, password}
  → handler 调用 userStore.CreateUser()
  → MySQL INSERT INTO users (username, password_hash, created_at)
  → 返回成功

登录流程（修改 Login handler）：
POST /api/login {username, password}
  → handler 调用 userStore.ValidateUser()
  → MySQL SELECT FROM users
  → bcrypt 验证
  → 记录日志：INSERT INTO login_logs (username, login_time, ip_address)
  → 返回 JWT token

Profile 查询（新增 handler）：
GET /api/user/profile (JWT)
  → 查询用户信息：SELECT username, created_at FROM users
  → 查询最近登录时间：SELECT login_time FROM login_logs ORDER BY login_time DESC
  → 查询登录列表：SELECT login_time FROM login_logs ORDER BY login_time DESC LIMIT 50
  → 返回 JSON
```

### 文件变更详细清单

```
[新增] internal/db/mysql.go
  - NewDB(dsn string) (*sql.DB, error)
  - InitTables(db *sql.DB) error

[重写] internal/user/store.go
  - 构造函数改为 NewStore(db *sql.DB) *Store
  - CreateUser: INSERT INTO users ... VALUES (?, ?, NOW())
  - ValidateUser: SELECT password_hash FROM users WHERE username = ?
  - GetUserInfo: SELECT username, created_at FROM users WHERE username = ?
  - RecordLogin: INSERT INTO login_logs (username) VALUES (?)
  - GetLoginLogs: SELECT login_time FROM login_logs WHERE username = ? ORDER BY login_time DESC LIMIT 50

[修改] internal/handler/auth_handler.go
  - handler 新增 userStore 引用
  - Login 成功后调用 RecordLogin 记录登录日志

[新增] internal/handler/profile_handler.go
  - Profile structure, NewProfileHandler, HandleProfile

[修改] cmd/server/main.go
  - 新增 -mysql-dsn 参数
  - 连接 MySQL，初始化表
  - 创建 userStore 改用 MySQL

[新增] static/profile.html — 用户主页
[新增] static/profile.js — 获取用户信息并渲染
[修改] static/index.html / editor.html — 右上角用户按钮
[修改] static/style.css — 用户按钮样式
[修改] go.mod — 添加 mysql driver
```

### Config 扩展

```go
type Config struct {
    Port       string
    StorageDir string
    MySQLDSN   string  // 新增
}
```

---

## 安全考虑

| 事项 | 处理方式 |
|------|---------|
| SQL 注入 | 全部使用参数化查询（`?` 占位符），禁止字符串拼接 |
| 密码存储 | bcrypt（不变） |
| DSN 密码 | 通过命令行参数传入，不硬编码 |
| 数据库连接 | 应用启动时连接，失败则报错退出 |

## 兼容性

- [x] 现有 handler 接口不变（`CreateUser(username, password)` 签名一致）
- [x] 现有 API 端点不变
- [x] 前端 js 模块结构不变
- [x] JWT 认证机制不变
