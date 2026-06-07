# 文档数据库存储 — 任务拆解

> **关联 Spec**: `docs/specs/文档数据库存储-spec.md`
> **关联技术方案**: `docs/specs/文档数据库存储-tech-design.md`

---

## 任务依赖关系

```
Task 1 (db/mysql.go 建表)
  └── Task 2 (docstore 包实现)
        └── Task 3 (FileService 改造)
              └── Task 4 (main.go 组装 + config)
                    └── Task 5 (更新 readme)
                          └── Task 6 (构建验证)
```

---

## Task 1: 数据库 — 在 InitTables 中新增 documents 表

**描述**：在 `internal/db/mysql.go` 的 `InitTables()` 函数中新增 `documents` 表的建表语句。

**接受标准**：
- [ ] `InitTables()` 执行后，数据库中存在 `documents` 表
- [ ] 表包含 id / username / filename / content / created_at / updated_at 字段
- [ ] 有 `(username, filename)` 唯一索引
- [ ] 有 `(username)` 普通索引
- [ ] 字符集 utf8mb4，引擎 InnoDB

**验证方式**：
- [ ] 启动项目，观察日志无建表错误
- [ ] 手动连接 MySQL：`DESC documents;` 确认表结构

**依赖**：无
**涉及文件**：
- `internal/db/mysql.go`

---

## Task 2: 存储层 — 实现 docstore 包

**描述**：创建 `internal/docstore/doc_store.go`，实现基于 MySQL 的文档 CRUD 操作。

包含以下方法：
- `NewStore(db *sql.DB) *Store`
- `ListFiles(username string) ([]string, error)` — 按文件名排序返回用户文件列表
- `ReadFile(username, filename string) (string, error)` — 读取文件内容，文件不存在返回哨兵错误
- `SaveFile(username, filename, content string) error` — 使用 INSERT … ON DUPLICATE KEY UPDATE
- `DeleteFile(username, filename string) error` — 删除文件，文件不存在返回哨兵错误

**接受标准**：
- [ ] 所有 SQL 使用参数化查询
- [ ] `SaveFile` 使用 upsert（INSERT … ON DUPLICATE KEY UPDATE）
- [ ] 文件不存在时返回 `ErrFileNotFound` 哨兵错误
- [ ] 导出符号有完整注释
- [ ] `go vet ./internal/docstore/...` 通过

**验证方式**：
- [ ] `go vet ./...` 无错误
- [ ] 手动调用测试（后续通过 API 验证）

**依赖**：Task 1
**涉及文件**：
- `internal/docstore/doc_store.go`（新增）

---

## Task 3: Service — 改造 FileService 使用 docstore

**描述**：修改 `internal/service/file_service.go`，将文件系统操作全部替换为 `docstore.Store` 调用。

**变更内容**：
- 构造函数：`NewFileService(baseDir string)` → `NewFileService(store *docstore.Store)`
- 移除字段：`baseDir` 和所有文件系统相关方法（`getUserDir`, `EnsureUserDir`, `safeFilePath`）
- 保留方法签名：`ListFiles`, `ReadFile`, `SaveFile`, `DeleteFile`（返回值和参数不变）
- 内部逻辑改为调用 `s.store.ListFiles(username)` 等

**接受标准**：
- [ ] 构造函数签名变更为接收 `*docstore.Store`
- [ ] 四个方法的参数和返回值与原来完全一致
- [ ] 不再依赖 `os` 包的文件操作
- [ ] `go vet ./...` 通过
- [ ] 编译成功

**验证方式**：
- [ ] `go build ./...` 无错误
- [ ] `go vet ./...` 无错误

**依赖**：Task 2
**涉及文件**：
- `internal/service/file_service.go`
- `internal/model/file.go`（可选，增加 Document 结构体）

---

## Task 4: 入口 — 更新 main.go 和 config

**描述**：
1. 在 `main.go` 中：创建 `docstore.Store` 并注入到 `FileService`
2. `config.go`：`StorageDir` 标注为可选

**main.go 变更**：
```go
// 修改前
fileService := service.NewFileService(cfg.StorageDir)

// 修改后
docStore := docstore.NewStore(database)
fileService := service.NewFileService(docStore)
```

同时可移除 `os.MkdirAll` 创建 storage 目录的逻辑（改为可选/注释）。

**接受标准**：
- [ ] 项目可正常启动，不再依赖 `storage` 配置指向的目录存在
- [ ] 启动日志正常，无 panic 或 Fatal

**验证方式**：
- [ ] `go build ./...` 成功
- [ ] 启动项目后访问 `http://localhost:3000` 正常

**依赖**：Task 3
**涉及文件**：
- `cmd/server/main.go`
- `internal/config/config.go`

---

## Task 5: 文档 — 更新 readme.md

**描述**：根据变更同步更新 `readme.md`。

**变更内容**：
- 目录结构中：在 `internal/` 下新增 `docstore/` 条目，从 `service/file_service.go` 说明中移除文件系统描述
- 数据库章节：新增 `documents` 表描述和 SQL
- 核心安全设计：移除路径遍历防护相关描述（不再需要）
- 如果 API 章节有文件操作说明，确认不变

**接受标准**：
- [ ] readme.md 反映了最新的项目结构
- [ ] 数据库章节包含 documents 表

**验证方式**：
- [ ] 预览 readme.md 内容通顺

**依赖**：Task 4
**涉及文件**：
- `readme.md`

---

## Task 6: 构建验证 — 编译并启动测试

**描述**：整合所有变更，编译并启动项目进行端到端验证。

**验证步骤**：
1. `go build ./...` — 确认编译通过
2. `go vet ./...` — 确认无 vet 警告
3. 启动项目
4. 注册账号 → 创建文档 → 读取文档 → 修改保存 → 删除文档
5. 验证登录/AI/用户主页不受影响

**接受标准**：
- [ ] 编译通过
- [ ] go vet 无错误
- [ ] 4 个文档 CRUD API 全部正常
- [ ] 其他功能（登录、AI 对话等）不受影响

**依赖**：Task 5
**涉及文件**：无（纯验证）
