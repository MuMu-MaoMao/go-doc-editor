# Spec：AI 角色扮演功能

- **版本**: v1.0
- **日期**: 2026-06-06
- **关联需求日志**: [2026-06-06-feat-ai-角色扮演-需求.md](../logs/功能日志/2026-06-06-feat-ai-角色扮演-需求.md)

---

## Objective（目标）

在 AI 对话面板中新增角色扮演功能，用户可以从预设角色列表中选择一个，
AI 以该角色的身份和语气对文档内容进行评价和建议。

**用户故事**：
> 作为一个文档编辑者，我想选择 AI 扮演"专业编辑"角色来审阅我的文档，
> 或者选择"友善读者"来获得第一印象反馈，从而从不同角度改进我的文档。

**成功标准**：
- 每个角色具有不同的 system prompt，AI 回应风格有明显差异
- 角色选择 UI 直观，用户无需学习即可使用
- 新增角色无需修改前端代码（后端定义）

## Tech Stack（技术栈）

本项目现有技术栈：
- **后端**: Go 1.25.1（标准库，无第三方 Web 框架）
- **AI**: DeepSeek API（deepseek-chat 模型）
- **前端**: 原生 HTML/CSS/JavaScript（ES Modules）
- **认证**: JWT（`github.com/golang-jwt/jwt/v5`）

本次新增不需要额外依赖。

## Project Structure（项目结构变动）

本次功能涉及以下文件的修改：

```
internal/
├── service/ai_service.go          # [修改] 添加角色定义数据结构
├── handler/ai_handler.go          # [修改] 添加 /api/ai/roles 端点
cmd/server/main.go                 # [修改] 注册新路由
static/
├── ai-chat.js                     # [修改] 角色选择逻辑、消息构建
├── editor.html                    # [修改] 添加角色选择 UI
└── style.css                      # [修改] 角色选择器样式
```

### 角色数据结构（后端）

```go
// Role 定义 AI 角色的身份和说话风格
type Role struct {
    ID          string `json:"id"`          // 唯一标识，如 "professional-editor"
    Name        string `json:"name"`        // 显示名称，如 "专业编辑"
    Description string `json:"description"`  // 简短描述，如 "帮你审校语法、优化句式"
    SystemPrompt string `json:"systemPrompt"` // 对应的 system message
}
```

### 预设角色定义

```go
var DefaultRoles = []Role{
    {
        ID:          "professional-editor",
        Name:        "专业编辑",
        Description: "审校语法、优化句式结构",
        SystemPrompt: "你是一位经验丰富的专业编辑，擅长审校文档的语法、句式、结构。\
请从编辑的角度对文档内容提供反馈：指出语法错误、优化句式结构、\
改进用词准确性。请用专业但友好的语气回复。",
    },
    {
        ID:          "humorous-writer",
        Name:        "幽默文案师",
        Description: "用幽默的方式点评内容",
        SystemPrompt: "你是一位以幽默风趣著称的文案师，善于用轻松诙谐的方式\
点评文档内容。你的反馈应该让人会心一笑，但同时提供有建设性的意见。\
请用幽默的语气回复，适当使用比喻和夸张手法。",
    },
    {
        ID:          "strict-mentor",
        Name:        "严格导师",
        Description: "高标准严要求，指出所有不足",
        SystemPrompt: "你是一位要求严格的导师，对文档质量有极高的标准。\
你会毫不留情地指出文档中的所有问题和不足，包括逻辑漏洞、表达不清、\
论据不足等。你的目标是帮助用户达到最高标准。请用严肃、直接的语气回复。",
    },
    {
        ID:          "friendly-reader",
        Name:        "友善读者",
        Description: "以普通读者的视角给出第一印象",
        SystemPrompt: "你是一位友善的普通读者，从阅读体验的角度对文档\
给出第一印象反馈。你会告诉用户文档读起来感觉如何，哪些部分吸引人，\
哪些部分需要更清晰的解释。请用温和、鼓励的语气回复。",
    },
    {
        ID:          "consultant",
        Name:        "行业顾问",
        Description: "从行业专业角度给出建议",
        SystemPrompt: "你是一位资深的行业顾问，具有丰富的领域知识。\
你会从专业角度分析文档内容，提供行业层面的建议和见解。\
指出文档中对行业理解不准确的地方，并提供改进建议。\
请用专业、严谨但平易近人的语气回复。",
    },
}
```

## Code Style（代码风格）

### 后端新增代码风格

```go
// ✅ 角色定义使用常量风格的包级变量
var DefaultRoles = []Role{...}

// ✅ API 处理器遵循已有模式
func (h *AIHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
    // 返回角色列表，无需认证（仅包含公开信息）
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "roles":   service.DefaultRoles,
    })
}
```

### 前端新增代码风格

```javascript
// ✅ 遵循已有模块化模式，ES Module 导出
// ✅ 新增变量集中在 ai-chat.js 文件头部
let currentRole = null;
let roles = [];

// ✅ DOM 操作遵循已有模式
chatRoleSelect.addEventListener('change', (e) => {
    switchRole(e.target.value);
});
```

## Testing Strategy（测试策略）

本次功能测试策略：

| 测试类型 | 内容 | 方式 |
|---------|------|------|
| **后端单元测试** | 验证角色定义完整性 | `go test` |
| **前端手动测试** | UI 交互、角色切换显示 | 浏览器手动检查 |
| **AI 效果验证** | 不同角色回复风格差异 | 实际发送消息对比 |

测试关注点：
1. `/api/ai/roles` 返回正确的角色列表 JSON
2. 切换角色后 system prompt 正确更新
3. 发送消息时携带当前角色的 system prompt
4. 清空对话时保持当前角色选择

## Boundaries（边界条件）

### Always（总是做）
- 角色定义数据在后端维护
- 新角色只需后端添加一个 Role 结构体
- 切换角色即更新 system prompt
- 表单验证使用已有模式

### Ask first（先问后做）
- 修改预设角色的 system prompt 内容
- 添加需要前端 UI 变化的角色类型

### Never（不要做）
- 不在前端硬编码角色 system prompt
- 不删除默认角色（可禁用，不可删除）
- 不将角色选择状态发送到后端作为认证凭据

## Success Criteria（成功标准）

- [ ] 每个角色都有不同的 name、description、systemPrompt
- [ ] 前端展示角色选择器（工具栏按钮或下拉框）
- [ ] 选择角色后，system prompt 自动替换 messages[0]
- [ ] 切换角色时，对话历史被清空（因为上下文变了）
- [ ] 发送聊天请求时，携带当前角色的 system prompt
- [ ] 重启页面后，默认选中第一个角色
- [ ] 代码通过 `go vet ./...` 无错误
- [ ] 所有现有功能不受影响

## Open Questions（待定）

无（需求已明确）
