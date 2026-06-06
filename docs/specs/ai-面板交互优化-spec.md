# Spec：AI 面板交互优化

- **版本**: v1.0
- **日期**: 2026-06-06
- **关联需求日志**: [2026-06-06-feat-ai-面板交互优化-需求.md](../logs/功能日志/2026-06-06-feat-ai-面板交互优化-需求.md)

---

## Objective（目标）

对 AI 对话面板的三项交互体验进行优化，使之更符合桌面应用的操作习惯。

**用户故事**：
> 作为一个频繁使用 AI 助手的文档编辑者，我希望可以自由拖动 AI 面板的位置、轻松调整面板大小、并且角色选择器能融入工具栏而不是单独占一行。

**成功标准**：
- 面板可自由拖拽
- 面板可通过任意边框拉伸
- 角色选择器与附带文档按钮风格统一、并列显示

## Tech Stack（技术栈）

纯前端改动，不涉及后端：
- **HTML**: `static/editor.html` — AI 面板 DOM 结构调整
- **CSS**: `static/style.css` — 拖拽、拉伸、角色选择器样式
- **JS**: `static/ai-chat.js` — 拖拽逻辑、拉伸逻辑、角色选择器交互

无需新增依赖。

## Project Structure（项目结构变动）

```
static/
├── editor.html     # [修改] 移除 .ai-chat-role-bar，角色选择器移到 .ai-chat-input-toolbar
├── ai-chat.js      # [修改] 新增拖拽逻辑、边框拉伸逻辑、角色选择器按钮交互
└── style.css       # [修改] 新增拖拽/拉伸样式、角色选择器按钮组样式
```

### 交互行为定义

#### 拖拽（Drag）
- **拖拽手柄**: `.ai-chat-header` 整个头部区域
- **触发**: mousedown 在 header 上（排除按钮区域）
- **过程**: mousemove 时更新面板 `left`/`top`
- **结束**: mouseup 停止跟踪
- **边界**: 阻止面板完全拖出视口（至少保留 100px 可见）

#### 边框拉伸（Resize）
- **检测区域**: 面板边缘 8px 范围
- **光标反馈**: 自动切换 - `n-resize` `s-resize` `e-resize` `w-resize` `ne-resize` `nw-resize` `se-resize` `sw-resize`
- **触发**: mousedown 在边缘区域
- **过程**: mousemove 根据拉伸方向调整宽高和位置
- **结束**: mouseup 停止跟踪
- **约束**: min-width: 320px, min-height: 400px, max-width: 750px, max-height: 90vh

#### 角色选择器
- **外观**: 按钮形式，与"附带文档"按钮等高同宽风格
- **显示内容**: `🎭 {当前角色名} ▼`
- **点击**: 弹出下拉列表，点击选项切换角色
- **下拉列表**: 浮动层，每个选项显示角色名 + 描述

## Code Style（代码风格）

### CSS 新增样式风格

```css
/* 按钮组工具栏 */
.ai-chat-input-toolbar {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
}

/* 统一的工具栏按钮 */
.toolbar-btn {
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 6px 14px;
    font-size: 0.8rem;
    cursor: pointer;
    transition: all 0.2s;
    color: var(--text-secondary);
    white-space: nowrap;
    display: inline-flex;
    align-items: center;
    gap: 4px;
}

.toolbar-btn:hover {
    background: var(--bg-hover);
    border-color: var(--border-dark);
}

.toolbar-btn.active {
    background: #1a1a1a;
    color: white;
    border-color: #1a1a1a;
}
```

### JS 新增代码风格

```javascript
// 拖拽：遵循"变量声明在模块顶部、事件绑定在 DOM 就绪后"的模式
let isDragging = false;
let dragOffsetX = 0, dragOffsetY = 0;

// 拉伸：状态变量简洁清晰
let isResizing = false;
let resizeDirection = '';
let resizeStartX, resizeStartY, resizeStartW, resizeStartH;
```

## Testing Strategy（测试策略）

| 测试类型 | 内容 | 方式 |
|---------|------|------|
| **拖动测试** | 按住头部拖动到不同位置 | 手动浏览器测试 |
| **边框拉伸测试** | 分别拖动上/下/左/右/四角 | 手动浏览器测试 |
| **边界测试** | 拖出视口边缘、缩到最小 | 手动浏览器测试 |
| **角色选择器测试** | 点击切换、查看下拉列表 | 手动浏览器测试 |
| **回归测试** | 聊天、附带文档不受影响 | 手动浏览器测试 |

## Boundaries（边界条件）

### Always
- 拖拽时鼠标移到 header 外仍保持拖拽（松开才停止）
- 拉伸时鼠标移到面板外仍保持拉伸
- 角色选择器按钮在移动端也能正常展开

### Ask first
- 保存面板位置到 localStorage（后续优化）

### Never
- 不打断正常聊天交互
- 不修改发送消息的流程
- 不依赖第三方库（全部原生 JS 实现）

## Success Criteria

- [ ] 按住 header 任意位置（按钮除外）可拖动面板
- [ ] 面板拖拽不超出视口边界
- [ ] 鼠标移到面板边缘自动切换光标样式
- [ ] 按住上/下/左/右/四角均可拉伸
- [ ] 拉伸时保持最小/最大尺寸限制
- [ ] 角色选择器按钮与"附带文档"按钮在同一行
- [ ] 角色选择器按钮显示当前角色名 + ▼
- [ ] 点击角色选择器按钮弹出下拉列表
- [ ] 选中角色后按钮文本更新
- [ ] 所有现有聊天功能正常

## Open Questions

无（需求已明确）
