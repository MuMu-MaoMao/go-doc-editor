# 文档数据库存储 — 技术方案设计

> **关联 Spec**: `docs/specs/文档数据库存储-spec.md`
> **关联 ADR**: `docs/adr/ADR-005-document-db-storage.md`

---

## 1. 影响文件清单

### 1.1 新增文件

| 文件 | 说明 |
|------|------|
| `internal/docstore/doc_store.go` | 文档数据库存储模块：封装所有文档 CRUD 的 SQL 操作 |

### 1.2 修改文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/db/mysql.go` | 修改 | `InitTables()` 新增 `documents` 表的建表语句 |
| `internal/service/file_service.go` | 重写 | 构造函数改为接收 `*docstore.Store`，方法委托给 docstore |
| `internal/model/file.go` | 微调 | 新增 `Document` 结构体（仅内部使用），`Response` 保持不变 |
| `cmd/server/main.go` | 修改 | 创建 `docstore.Store` 并注入到 `FileService` |
| `internal/config/config.go` | 微调 | `StorageDir` 标注为可选字段 |
| `docs/adr/ADR-005-document-db-storage.md` | 新增 | 架构决策记录 |
| `readme.md` | 修改 | 更新目录结构和数据库章节 |

### 1.3 不受影响文件

前端的 `static/` 目录、`handler/` 层（接口不变）、`middleware/`、`auth/`、`user/`、`ai_service.go` 等完全不变。

---

## 2. 数据库表设计

### documents 表

```sql
CREATE TABLE IF NOT EXISTS documents (
    id         BIGINT       AUTO_INCREMENT PRIMARY KEY COMMENT '文档唯一标识',
    username   VARCHAR(255) NOT NULL        COMMENT '所属用户名，与 users 表关联',
    filename   VARCHAR(255) NOT NULL        COMMENT '文件名（含扩展名）',
    content    LONGTEXT                     COMMENT '文档内容，使用 LONGTEXT 支持大文件',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    UNIQUE INDEX idx_documents_user_file (username, filename) COMMENT '同一用户下文件名唯一',
    INDEX idx_documents_username (username) COMMENT '按用户查询文档列表'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文档存储表';
```

### 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| content 类型 | `LONGTEXT`（最大 4GB） | 文档编辑器可能处理长文本，TEXT（64KB）可能不够 |
| 唯一约束 | `UNIQUE(username, filename)` | 同一用户不能有两个同名文件，替代文件系统的同名覆盖 |
| 更新时间 | `ON UPDATE CURRENT_TIMESTAMP` | 自动维护，无需应用层处理 |
| 字符集 | `utf8mb4` | 与项目其他表一致，支持完整 Unicode（包括 emoji） |
| 引擎 | `InnoDB` | 事务支持、行级锁，与项目其他表一致 |

---

## 3. 数据流

### 迁移后调用链路

```
HTTP 请求
  → handler/file_handler.go (解析请求、提取用户名)
    → service/file_service.go (业务逻辑，不变)
      → docstore.Store (数据库 CRUD)
        → MySQL (documents 表)
```

### 各操作的数据流

#### 列出文件 (GET /api/files)
```
handler.ListFiles
  → fileService.ListFiles(username)
    → docstore.ListFiles(username)
      → SELECT filename FROM documents WHERE username = ? ORDER BY filename
    ← []string{"a.md", "b.md"}
  ← Response{Success: true, Files: names}
```

#### 读取文件 (GET /api/file/{filename})
```
handler.ReadFile
  → fileService.ReadFile(username, filename)
    → docstore.ReadFile(username, filename)
      → SELECT content FROM documents WHERE username = ? AND filename = ?
    ← "文件内容"
  ← Response{Success: true, Content: content}
```

#### 保存文件 (POST /api/file/{filename})
```
handler.SaveFile
  → fileService.SaveFile(username, filename, content)
    → docstore.SaveFile(username, filename, content)
      → INSERT ... ON DUPLICATE KEY UPDATE (UPSERT)
        → 已存在: UPDATE content, updated_at
        → 不存在: INSERT new row
    ← nil
  ← Response{Success: true, Message: "文件保存成功"}
```

#### 删除文件 (DELETE /api/file/{filename})
```
handler.DeleteFile
  → fileService.DeleteFile(username, filename)
    → docstore.DeleteFile(username, filename)
      → DELETE FROM documents WHERE username = ? AND filename = ?
    ← nil (或"文件不存在"错误)
  ← Response{Success: true, Message: "文件删除成功"}
```

---

## 4. 方案对比

### 方案 A：新增 docstore 包 + FileService 委托（推荐 ✓）

**描述**：新增 `internal/docstore` 包封装数据库操作，`FileService` 构造函数接受 `*docstore.Store`，
方法内部委托给 docstore。

```
FileService ──委托──→ docstore.Store ──SQL──→ MySQL
```

| 维度 | 评估 |
|------|------|
| 文件改动 | 新增 1 文件，修改 4 文件 |
| handler 层改动 | **无** |
| 测试能力 | docstore 可独立测试，FileService 可 mock docstore |
| 未来扩展 | 如需换存储方式，替换 docstore 实现即可 |
| 架构一致性 | 与 `user.Store` 模式一致（数据库操作独立成 store） |

### 方案 B：直接在 FileService 中使用 *sql.DB

**描述**：不新增独立包，直接在 `FileService` 中保存 `*sql.DB` 引用，所有 SQL 直接在 `file_service.go` 中操作。

| 维度 | 评估 |
|------|------|
| 文件改动 | 只改 file_service.go 一个文件 |
| 架构 | FileService 同时负责业务逻辑和数据库操作，违反单一职责 |
| 测试 | 难以独立测试数据库逻辑 |
| 与现有风格 | 与 `user.Store` 模式不一致 |

### 方案 C：移除 FileService，handler 直接调用 docstore

**描述**：FileService 太薄被移除，handler 层直接调用 docstore 的方法。

| 维度 | 评估 |
|------|------|
| 文件改动 | 删除 file_service.go，修改 file_handler.go 和 main.go |
| 架构 | 打破现有分层结构 |
| handler 职责 | handler 直接操作存储层，未来增加业务逻辑会导致 handler 膨胀 |

### 选择方案 A 的理由

1. **关注点分离**：存储逻辑独立成包，与业务逻辑解耦
2. **与现有风格一致**：`user.Store` 采用同样的模式
3. **handler 层零改动**：降低风险
4. **可测试性**：docstore 可独立测试
5. **可扩展性**：未来如需添加版本管理、全文搜索等功能，可以在 docstore 中增加方法

---

## 5. docstore.Store API 设计

```go
// Package docstore 提供基于 MySQL 的文档 CRUD 存储。
// 所有操作基于用户名进行用户隔离，通过参数化查询防止 SQL 注入。
package docstore

// Store 提供文档的数据库 CRUD 操作。
type Store struct {
    db *sql.DB
}

// NewStore 创建基于 MySQL 的文档存储。
func NewStore(db *sql.DB) *Store

// ListFiles 返回用户的所有文档文件名列表，按文件名排序。
func (s *Store) ListFiles(username string) ([]string, error)

// ReadFile 读取用户指定文件的内容。
// 文件不存在返回 ErrFileNotFound。
func (s *Store) ReadFile(username, filename string) (string, error)

// SaveFile 保存/覆盖用户指定文件。
// 使用 INSERT … ON DUPLICATE KEY UPDATE 实现 upsert 语义：
// - 如果文件已存在，更新内容和 updated_at
// - 如果文件不存在，创建新记录
func (s *Store) SaveFile(username, filename, content string) error

// DeleteFile 删除用户指定文件。
// 文件不存在返回 ErrFileNotFound。
func (s *Store) DeleteFile(username, filename string) error
```

---

## 6. FileService 变更

```go
// 修改前
type FileService struct {
    baseDir string
}
func NewFileService(baseDir string) *FileService

// 修改后
type FileService struct {
    store *docstore.Store
}
func NewFileService(store *docstore.Store) *FileService
```

方法签名保持不变（`ListFiles`, `ReadFile`, `SaveFile`, `DeleteFile`），
内部实现改为调用 `s.store` 对应方法。

**移除了：**
- `baseDir` 字段
- `getUserDir()` 方法
- `EnsureUserDir()` 方法
- `safeFilePath()` 方法
- 所有 `os.*` 文件系统操作

---

## 7. main.go 变更

```go
// 修改前
fileService := service.NewFileService(cfg.StorageDir)

// 修改后
docStore := docstore.NewStore(database)
fileService := service.NewFileService(docStore)
```

---

## 8. 安全审计

| 风险点 | 缓解措施 |
|--------|---------|
| SQL 注入 | 全部使用参数化查询（`?` 占位符），无字符串拼接 |
| 跨用户访问 | 所有查询 WHERE 条件包含 `username = ?`，按用户过滤 |
| 文件名特殊字符 | 直接存储在数据库字段中，无文件系统路径限制 |
| 超大数据内容 | LONGTEXT 支持最大 4GB，无应用层限制（受 MySQL max_allowed_packet 限制） |

---

## 9. 向后兼容性

| 维度 | 兼容性 |
|------|--------|
| API 接口 | 完全向后兼容，请求/响应格式不变 |
| 配置文件 | `storage` 字段暂保留，标记为可选；不破坏现有 `config.json` |
| 现有数据 | 本次不做数据迁移，文件系统上的历史文档不会自动导入数据库 |
| 前端 | 完全不变 |

---

## 10. 与现有 ADR 的关联

- 本方案的架构模式与 `user.Store`（数据库操作独立成 store）一致
- 新的 `docstore` 包遵循与 `user` 包相同的模式

---

## 11. 开放问题

- **数据迁移**：是否需要提供从文件系统到数据库的数据迁移脚本？→ **本次不做**，先保证新功能正常运行
- **文件内容大小限制**：MySQL `max_allowed_packet` 默认 64MB，是否需要配置文件可调？→ **暂不需要**，使用默认值
