# 技术方案设计：AI 面板交互优化

- **关联 Spec**: [ai-面板交互优化-spec.md](./ai-面板交互优化-spec.md)
- **日期**: 2026-06-06

---

## 需求一：按住对话框上部拖动

### 方案对比

**方案 A：原生 JS 拖拽（选中 ✅）**

思路：监听 header 上的 mousedown → mousemove → mouseup，更新面板的 `left`/`top`。

```
mousedown on header
  → 记录鼠标位置与面板左上角的偏移量
  → 切换到 position: fixed 的 left/top 坐标系（从 right/bottom 切换）
  → 标记 isDragging = true

mousemove on document
  → 如果 isDragging
  → 计算 newLeft = mouseX - offsetX
  → 计算 newTop = mouseY - offsetY
  → 边界约束（至少 100px 在视口内）
  → 更新 panel.style.left / panel.style.top

mouseup on document
  → isDragging = false
```

**方案 B：HTML5 Drag API**
- ❌ HTML5 Drag API 设计用于拖放操作（拖到目标区域），不适合自由拖拽

**方案 C：CSS `position: sticky` + 滚动**
- ❌ 与固定定位冲突，无法实现自由拖拽

### 选中方案

**选中方案 A**：原生 JS 实现，轻量可控。

**关键设计决定**：拖拽开始时将面板坐标系从 `right/bottom` 切换到 `left/top`。
因为拖拽需要精确控制左/上坐标，而 CSS 的 right/bottom 在拖拽过程中难以计算。

---

## 需求二：按住边框拉伸大小

### 方案对比

**方案 A：自定义 JS 边框检测（选中 ✅）**

思路：通过 mousemove 检测鼠标是否在面板边缘 8px 范围内，
自动切换光标样式；mousedown 时记录方向开始拉伸。

```
mousemove on panel
  → 检测鼠标距离各边的距离
  → < 8px 时：切换光标（n/s/e/w/ne/nw/se/sw）
  → 记录当前鼠标所在边

mousedown on edge
  → 记录方向（top/bottom/left/right/组合）
  → 记录起始鼠标位置 + 面板起始宽高 + 面板起始位置
  → isResizing = true

mousemove on document
  → 如果 isResizing
  → 根据方向和鼠标移动量计算新尺寸和位置
  → 应用最小/最大尺寸限制
  → 更新面板

mouseup on document
  → isResizing = false
  → 恢复光标
```

**方案 B：创建 8 个不可见 handle 元素**
- 优点：利用元素事件，检测精确
- 缺点：增加 DOM 复杂度，多出 8 个元素
- → ❌ 方案 A 更简洁，无需额外 DOM

**方案 C：使用 CSS `resize` 属性（现有方案）**
- 缺点：只能在右下角拉伸，操作区域小
- → ❌ 不能满足需求

### 选中方案

**选中方案 A**：纯 JS 边界检测，无需额外 DOM 元素。

**关键设计决定**：
1. 拉伸方向动态判断（检测鼠标距四边的距离）
2. 同时支持四边和四角拉伸
3. 拉伸时同步更新面板尺寸（width/height）和位置（left/top）
   - 拖左边界 → 改变 left + width
   - 拖上边界 → 改变 top + height
   - 拖右边界 → 只改变 width
   - 拖下边界 → 只改变 height
4. 使用 `document` 级事件，防止鼠标移出面板时丢失事件

### 光标映射

| 方向 | 光标 | CSS cursor |
|------|------|-----------|
| 上 | ↑ | `n-resize` |
| 下 | ↓ | `s-resize` |
| 左 | ← | `w-resize` |
| 右 | → | `e-resize` |
| 左上 | ↖ | `nw-resize` |
| 右上 | ↗ | `ne-resize` |
| 左下 | ↙ | `sw-resize` |
| 右下 | ↘ | `se-resize` |

---

## 需求三：角色选择器 UI 美化

### 方案对比

**方案 A：自定义下拉按钮组件（选中 ✅）**

移除原有 `<select>` 和 `.ai-chat-role-bar` 区域。

用自定义按钮 + 浮动下拉列表替代：

```
[📎 附带文档] [🎭 专业编辑 ▼]    ← 同一行，风格一致
                    │
                    ├── 🎭 专业编辑
                    ├── 😄 幽默文案师
                    ├── 📐 严格导师
                    ├── 📖 友善读者
                    └── 💼 行业顾问
```

交互：
- 默认显示当前角色名 + ▼ 箭头
- 点击按钮展开/收起下拉列表
- 点击选项切换角色，同时关闭下拉列表
- 点击外部关闭下拉列表

**关键设计决定**：
- 使用 `role-emoji` 字段为每个角色配一个图标（在 HTML 或 JS 中定义）
- 下拉列表为绝对定位，z-index 高于面板
- 选择后按钮文本更新为 `🎭 角色名 ▼`

**方案 B：只改 CSS 样式不换元素**
- 可以修改 `<select>` 的外观，但下拉菜单的样式无法完全控制
- → ❌ 无法与"附带文档"按钮完全统一风格

### 选中方案

**选中方案 A**：自定义下拉按钮组件，完全控制外观。

---

## 文件变更清单总览

```
[修改] static/editor.html
  - 移除 .ai-chat-role-bar（及其内部的 select）
  - 在 .ai-chat-input-toolbar 中添加角色下拉按钮的占位 DOM

[修改] static/style.css
  - 新增拖拽相关的光标、状态样式
  - 新增自定义下拉菜单样式（浮动层）
  - 统一工具栏按钮样式（.toolbar-btn）
  - 移除 .ai-chat-role-bar 相关样式
  - AI 面板增加 overflow: hidden 不影响拖拽

[修改] static/ai-chat.js
  - 新增拖拽逻辑（~40 行）
  - 新增边框拉伸逻辑（~80 行）
  - 新增自定义角色选择器交互逻辑（~30 行）
  - 移除原有的 select 事件绑定
  - 角色切换函数 switchToRole 保持接口不变
```

## 兼容性

- 所有现有功能向后兼容 ✅
- 聊天/附带文档/角色切换逻辑不变 ✅
- 无新增外部依赖 ✅
