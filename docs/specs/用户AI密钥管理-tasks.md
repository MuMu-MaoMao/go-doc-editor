# 任务拆解：用户 AI-Key 管理系统

- **日期**: 2026-06-06

---

## 依赖图谱

```
Task 1: 数据库新增 ai_keys 表
   │
   ├── Task 2: user.Store 新增 AIKey CRUD 方法
   │      │
   │      ├── Task 3: 新增 AIKey handler（CRUD 端点）
   │      │
   │      ├── Task 4: 重构 ai_service.go（动态参数）
   │      │
   │      └── Task 5: 修改 ai_handler.go（从数据库获取用户 Key）
   │             │
   │             └── Task 6: 修改 profile_handler（返回 Key 列表）
   │                    │
   │                    └── Task 7: 前端 profile 页面改造
```

## 任务列表

### Task 1：数据库 - 新增 ai_keys 表

**文件**: `internal/db/mysql.go`
- InitTables 新增 ai_keys 表

### Task 2：后端 - user.Store 新增 AIKey CRUD

**文件**: `internal/user/store.go`
- 新增 `AIKey` 结构体
- `GetUserAIKeys(username) ([]AIKey, error)`
- `CreateAIKey(username, name, key, url, model) (*AIKey, error)`
- `ActivateAIKey(username, keyID) error`
- `DeleteAIKey(username, keyID) error`
- `GetActiveAIKey(username) (*AIKey, error)`

### Task 3：后端 - AIKey handler

**文件**: `internal/handler/profile_handler.go`
- 新增 `GET /api/user/ai-keys`
- 新增 `POST /api/user/ai-keys`
- 新增 `PUT /api/user/ai-keys/{id}/activate`
- 新增 `DELETE /api/user/ai-keys/{id}`

### Task 4：重构 ai_service.go

**文件**: `internal/service/ai_service.go`
- `ChatStream` 增加 `apiKey, apiURL, model` 参数
- 移除全局 `deepseekAPIURL` 常量
- 移除 `AIService.apiKey` 字段（不再需要）

### Task 5：修改 ai_handler.go

**文件**: `internal/handler/ai_handler.go`
- `Chat` 从数据库查询用户激活的 Key
- 无激活 Key 时回退全局配置

### Task 6：修改 profile_handler.go

**文件**: `internal/handler/profile_handler.go`
- profile 响应中加入 AI key 列表

### Task 7：前端 profile 页面

**文件**: `static/profile.html` + `static/profile.js`
- 新增"AI 配置"区域
- Key 列表显示
- 新增 Key 表单（名称 + API-Key + URL + 模型选择）
- 激活/删除操作
- 模型选项：DeepSeek-Flash / DeepSeek-Pro

### Task 8：更新 main.go

**文件**: `cmd/server/main.go`
- 注册 AIKey 路由
- 传递 userStore 给 aiHandler
