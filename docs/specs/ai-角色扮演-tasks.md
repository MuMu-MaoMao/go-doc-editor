# 任务拆解：AI 角色扮演

- **关联设计**: [ai-角色扮演-tech-design.md](./ai-角色扮演-tech-design.md)
- **日期**: 2026-06-06

---

## 依赖图谱

```
Task 1: 后端角色数据结构
    │
    ├── Task 2: /api/ai/roles 端点
    │
    ├── Task 3: 前端获取并渲染角色列表
    │       │
    │       └── Task 4: 角色选择交互逻辑
    │
    └── Task 5: 切换角色清空对话 + 更新 system prompt
            │
            └── Task 6: CSS 样式 + 路由注册

Task 7: 验证与测试
```

注意：Task 1→2→3→4→5→6→7 是顺序依赖关系，
每个 Task 完成后前端的角色列表 UI 会逐步可用。

## 任务列表

### Phase 1：后端基础

---

#### Task 1：后端角色数据结构

**描述**：在 `ai_service.go` 中定义 Role 结构体和预设角色列表。

**接受标准**：
- [ ] Role 结构体包含 ID、Name、Description、SystemPrompt 字段
- [ ] DefaultRoles 至少定义 5 个预设角色
- [ ] 每个角色有不同的 system prompt

**验证方式**：
- [ ] `go vet ./...` 通过
- [ ] 直接检查 DefaultRoles 变量定义

**涉及文件**：
- `internal/service/ai_service.go`

**预计范围**：小（1 个文件，+40 行）

---

#### Task 2：新增 /api/ai/roles 端点

**描述**：在 `ai_handler.go` 中添加 ListRoles 处理器，返回角色列表 JSON。

**接受标准**：
- [ ] `GET /api/ai/roles` 返回 `{"success": true, "roles": [...]}`
- [ ] 每个角色包含 id, name, description（不返回 systemPrompt，前端仅用于展示）

**验证方式**：
- [ ] `go vet ./...` 通过
- [ ] 检查 handler 代码结构是否正确

**涉及文件**：
- `internal/handler/ai_handler.go`

**预计范围**：小（1 个文件，+15 行）

---

### Checkpoint：后端基础
- [ ] `go vet ./...` 通过
- [ ] 代码结构正确

---

### Phase 2：路由注册 + 前端基础

---

#### Task 3：路由注册 + 前端获取角色列表

**描述**：在 `main.go` 注册 `/api/ai/roles` 路由；在前端 `ai-chat.js` 中添加初始化时 fetch 角色列表的逻辑。

**接受标准**：
- [ ] `/api/ai/roles` 路由已注册（无需 JWT 认证）
- [ ] `ai-chat.js` 启动时自动请求角色列表
- [ ] 角色数据保存在 `roles` 变量中

**验证方式**：
- [ ] `go vet ./...` 通过
- [ ] 检查前端 console 确认请求正常

**涉及文件**：
- `cmd/server/main.go`（+3 行）
- `static/ai-chat.js`（+15 行）

**预计范围**：小（2 个文件）

---

#### Task 4：前端角色选择 UI

**描述**：在 `editor.html` 的 AI 面板头部添加角色下拉选择器，添加对应的 CSS 样式。

**接受标准**：
- [ ] AI 面板头部显示 `<select>` 下拉框
- [ ] 角色列表动态渲染（从 `roles` 变量读取）
- [ ] 选中一个角色时触发自定义事件或调用切换函数
- [ ] 下拉框样式与现有 UI 风格一致

**验证方式**：
- [ ] 手动检查编辑器页面 AI 面板

**涉及文件**：
- `static/editor.html`（+10 行）
- `static/style.css`（+20 行）

**预计范围**：小（2 个文件）

---

### Phase 3：核心逻辑

---

#### Task 5：角色切换交互逻辑

**描述**：在 `ai-chat.js` 中，选择角色时更新 messages 数组的 system prompt，并清空对话历史。

**接受标准**：
- [ ] 切换角色后 messages[0]（system）被替换为新角色的 systemPrompt
- [ ] 切换角色后对话历史被清空（保留新的 system prompt）
- [ ] 切换角色后显示提示"已切换到 [角色名]"
- [ ] 默认选中第一个角色

**验证方式**：
- [ ] 手动检查切换角色后 AI 回复风格变化

**涉及文件**：
- `static/ai-chat.js`（+30 行）

**预计范围**：小（1 个文件）

---

### Checkpoint：功能完整
- [ ] 所有代码修改完成
- [ ] 角色选择器正常显示
- [ ] 切换角色后 AI 回复风格变化

---

### Phase 4：验证

---

#### Task 6：验证与测试

**描述**：运行完整测试、确认功能正常。

**验证方式**：
- [ ] `go vet ./...` 通过
- [ ] 手动确认 UI 正常
- [ ] 确认切换角色后 AI 回复风格不同

**不涉及文件改动**。

---

## 任务与提交计划

```
Commit 1 (Task 1): feat(service): add AI role data structures
Commit 2 (Task 2): feat(handler): add /api/ai/roles endpoint
Commit 3 (Task 3 + 4): feat(static): add role selector UI and routing
Commit 4 (Task 5): feat(static): implement role switch logic
Commit 5 (Task 6): test: verify role playing feature
```
