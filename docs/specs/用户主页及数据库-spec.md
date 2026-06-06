# Spec：用户主页 + MySQL 数据库

- **版本**: v1.0
- **日期**: 2026-06-06
- **关联需求**: [需求日志](../logs/功能日志/2026-06-06-feat-用户主页及数据库-需求.md)

---

## Objective

**三大目标**：
1. 用 MySQL 数据库替代 JSON 文件存储用户数据，增加注册时间和登录日志
2. 在页面右上角新增圆形用户按钮
3. 创建用户主页显示账户信息

**成功标准**：
- 用户注册/登录使用 MySQL，应用启动自动建表
- 每次登录记录时间和 IP
- 右上角按钮可见可点击
- 用户主页展示完整信息

## Tech Stack

| 技术 | 说明 |
|------|------|
| **Go MySQL 驱动** | `github.com/go-sql-driver/mysql` v1.8+ |
| **数据库** | MySQL 8.0.44 @ localhost:3306 |
| **数据库名** | `godoxedit` |
| **前端** | 原生 HTML/CSS/JS（ES Modules） |

## Project Structure

```
go-doc-editor/
├── internal/
│   ├── db/
│   │   └── mysql.go            # 🆕 MySQL 连接管理与表初始化
│   ├── user/
│   │   └── store.go            # [重写] 从 JSON → MySQL
│   └── handler/
│       ├── auth_handler.go     # [修改] Login 时记录登录日志
│       └── profile_handler.go  # 🆕 用户主页 API
├── cmd/server/main.go          # [修改] 添加 MySQL 依赖
├── static/
│   ├── index.html              # [修改] 右上角用户按钮
│   ├── editor.html             # [修改] 右上角用户按钮
│   ├── profile.html            # 🆕 用户主页
│   ├── profile.js              # 🆕 用户主页 JS
│   └── style.css               # [修改] 用户按钮样式
├── go.mod                      # [修改] 添加 MySQL 驱动
└── readme.md                   # [修改] 更新配置说明
```

## Database Schema

```sql
-- 用户表
CREATE TABLE IF NOT EXISTS users (
    username      VARCHAR(255) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 登录日志表
CREATE TABLE IF NOT EXISTS login_logs (
    id         BIGINT       AUTO_INCREMENT PRIMARY KEY,
    username   VARCHAR(255) NOT NULL,
    login_time DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_login_logs_username (username),
    INDEX idx_login_logs_time (login_time)
);
```

## API 接口

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | `/api/user/profile` | JWT | 返回用户名、注册时间、最近登录时间、登录列表 |

**响应格式**：
```json
{
    "success": true,
    "data": {
        "username": "testuser",
        "createdAt": "2026-06-06T10:00:00Z",
        "lastLoginAt": "2026-06-06T12:00:00Z",
        "loginLogs": [
            {"loginTime": "2026-06-06T12:00:00Z"},
            {"loginTime": "2026-06-06T11:00:00Z"}
        ]
    }
}
```

## Code Style

后端数据库操作风格：
```go
// 使用 database/sql 标准库
db, _ := sql.Open("mysql", dsn)

// 初始化表
func InitTables(db *sql.DB) error {
    _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ...`)
    return err
}
```

## Testing Strategy

| 类型 | 内容 |
|------|------|
| 启动测试 | 应用启动时自动建表 |
| API 测试 | 注册→登录→profile 接口链路 |
| 前端测试 | 按钮显示、跳转、页面数据展示 |

## Boundaries

**Always**:
- MySQL 连接配置通过命令行参数 `-mysql-dsn` 传入
- 表结构在应用启动时自动创建
- 密码仍使用 bcrypt

**Never**:
- 不直接在代码中硬编码数据库密码
- 不删除现有的 JSON 用户数据（留作备份）

## Success Criteria

- [ ] `go run cmd/server/main.go -mysql-dsn "..."` 正常启动并建表
- [ ] 注册用户成功写入 MySQL users 表
- [ ] 登录成功记录到 login_logs 表
- [ ] 右上角圆形按钮在 index 和 editor 均显示
- [ ] 点击按钮跳转到 profile.html
- [ ] profile 页面显示用户名、注册时间、最近登录时间、登录列表
- [ ] 登录时间按倒序排列
