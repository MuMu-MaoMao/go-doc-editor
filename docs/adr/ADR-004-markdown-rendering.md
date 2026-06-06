# ADR-004: 前端 Markdown 渲染与 XSS 防护方案

- **状态**: Accepted
- **提出日期**: 2026-05-20
- **提出人**: 项目初始团队

## 上下文

AI 响应内容包含 Markdown 格式（代码块、列表、标题等），
但当时前端仅以纯文本显示，无法渲染格式，影响阅读体验。
同时，AI 响应内容可能包含恶意 HTML，需要 XSS 防护。

## 考虑的方案

### 方案 A：marked + DOMPurify（选中）
- `marked` 库将 Markdown 解析为 HTML
- `DOMPurify` 清洗 HTML，防止 XSS
- 均为轻量级库，可通过 CDN 引入

### 方案 B：highlight.js + marked（备用）
- 在方案 A 基础上增加代码语法高亮
- 评估后认为代码高亮非必需，留作后续优化

### 方案 C：纯 CSS 渲染
- 无法正确渲染复杂 Markdown（表格、代码块等）
- 放弃

## 决策

选中**方案 A（marked + DOMPurify）**：

1. `marked` 提供标准的 Markdown 渲染
2. `DOMPurify` 提供可靠的 XSS 防护
3. 两者都通过 CDN 加载（`marked` + `DOMPurify`）

## 影响

### 正面影响
- AI 回复中的代码块、列表、标题等格式正确显示
- XSS 攻击得到有效防护
- 两者均为成熟库，社区广泛使用

### 负面影响
- 依赖 CDN 加载，离线场景不可用
- 增加页面加载体积（marked ~30KB + DOMPurify ~20KB）

## 关联
- 实现文档：`AI对话优化方案.md`（优化二）
- 涉及文件：`static/editor.html`、`static/ai-chat.js`、`static/style.css`
