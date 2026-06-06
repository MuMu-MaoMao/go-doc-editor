# 技术方案设计：AI 角色扮演

- **关联 Spec**: [ai-角色扮演-spec.md](./ai-角色扮演-spec.md)
- **日期**: 2026-06-06

---

## 方案对比

### 方案 A：后端定义角色 + 前端选择（推荐 ✅）

**角色数据流**：
```
初始化：
  前端 GET /api/ai/roles → 后端返回角色列表 → 前端渲染选择器

对话流程：
  用户选择角色 → 更新前端 messages[0]（system）
  → 用户发送消息 → POST /api/ai/chat {messages} 
  → 后端透传 → DeepSeek API → SSE 返回
```

**优点**：
- 添加角色只需改后端，前端自动获取
- API 密钥安全（仍在后端）
- 复用现有消息透传机制，无需改后端消息处理逻辑
- 新增角色零前端改动

**缺点**：
- 需要新增 API 端点 `GET /api/ai/roles`
- 前端需额外发一次请求获取角色列表

---

### 方案 B：前后端都定义角色

角色列表同时在前后端硬编码。

**优点**：
- 前端不依赖网络即可获取角色

**缺点**：
- 添加角色需要同时改两端，易不同步
- 违反 DRY 原则

→ **不推荐** ❌

---

### 方案 C：纯前端定义

所有角色 system prompt 写在前端 JavaScript 中。

**优点**：
- 实现最简单，无需新增 API

**缺点**：
- system prompt 暴露在前端代码中
- 后续可能包含敏感指令
- 增加角色需修改前端代码
- 不符合当前项目的安全规范（AI 密钥都在后端）

→ **不推荐** ❌

---

## 选中方案：方案 A 的详细设计

### 数据流

```
用户打开编辑器页面
    │
    ▼
editor.html 加载 → ai-chat.js 初始化
    │
    ├── GET /api/ai/roles（获取角色列表）
    │       │
    │       ▼
    │   后端 handler/ai_handler.go → ListRoles
    │       │  返回 JSON: {success: true, roles: [...]}
    │       ▼
    │   前端渲染角色选择器，默认选中第一个
    │
    ├── 用户选择角色
    │       │
    │       ▼
    │   更新 messages[0] = {role: "system", content: selectedRole.systemPrompt}
    │   清空对话历史（角色切换后旧上下文不适用）
    │   显示提示"已切换到 [角色名]"
    │
    └── 用户发送消息
            │
            ▼
        POST /api/ai/chat {messages: [...]}
            │   messages[0] 已经是当前角色的 system prompt
            ▼
        后端透传 → DeepSeek → SSE 返回
```

### 文件变更清单

```
变更清单：
  [修改] internal/service/ai_service.go
      - 新增 Role 结构体
      - 新增 DefaultRoles 变量
      - 新增 GetRoles() 方法

  [修改] internal/handler/ai_handler.go
      - 新增 ListRoles 处理函数
      - 新增 chatRequest 无需变更（复用现有结构）

  [修改] cmd/server/main.go
      - 注册路由: /api/ai/roles（注意：无需JWT认证，因为角色是公开信息）

  [修改] static/ai-chat.js
      - 初始化时 fetch 角色列表
      - 新增 roleSelector UI 绑定
      - 切换角色时更新 system prompt
      - 切换角色时清空对话

  [修改] static/editor.html
      - 在 AI 面板头部添加角色选择器（下拉框）

  [修改] static/style.css
      - 角色选择器样式

  [新增] docs/specs/ai-角色扮演-spec.md       （已完成）
  [新增] docs/specs/ai-角色扮演-tech-design.md  （本文档）
  [新增] docs/specs/ai-角色扮演-tasks.md        （下一步）
```

### 安全考虑

| 事项 | 处理方式 |
|------|---------|
| System prompt 内容 | 保存在后端，前端通过 API 获取只读副本 |
| JWT 认证 | 角色列表接口无需认证，但 AI 聊天接口仍需 JWT |
| XSS | 角色描述使用 textContent，不信任外部输入 |

### 兼容性检查

- backward compatible ✅ 不影响现有聊天功能
- 旧前端（不请求角色）仍可使用 ✅ 默认使用第一个角色

---

## 决策理由

选中**方案 A**，原因总结：
1. 符合项目现有架构（后端定义数据、前端展现）
2. 角色数据安全（不在前端暴露敏感 prompt）
3. 可扩展性强（新增角色只需后端一行代码）
4. 与现有消息透传机制完全兼容
