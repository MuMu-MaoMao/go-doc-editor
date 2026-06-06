# 02 · Git 分支策略

> go-doc-editor 采用 **GitHub Flow** 作为分支策略。
> 本文档说明分支模型、命名规则和合并流程。

---

## 2.1 为什么选择 GitHub Flow

| 考量 | 本项目的情况 |
|------|-------------|
| 团队规模 | 单人/小团队 |
| 发布频率 | 持续发布（随时可部署） |
| 版本需求 | 无需同时维护多个版本 |
| CI/CD | 必备，所有 PR 需通过检查 |

GitHub Flow 足够简单且高效，不需要 Git Flow 的复杂分支管理开销。

## 2.2 分支模型

```
main (长期分支，始终保持可部署)
  │
  ├── feat/xxx              # 新功能
  ├── fix/xxx               # 修复
  ├── refactor/xxx          # 重构
  ├── docs/xxx              # 文档
  └── chore/xxx             # 杂项（依赖、构建等）
```

## 2.3 核心规则

1. **`main` 分支始终保持可部署** — 任何时候 checkout main 都能直接编译运行。
2. **从 main 创建功能分支**。
3. **通过 Pull Request 合并回 main** — 禁止直接 push 到 main。
4. **PR 合并前必须通过 CI 检查**（测试、lint 等）。
5. **合并后立即删除远程功能分支**。

## 2.4 分支命名规范

| 类型 | 格式 | 示例 |
|------|------|------|
| 功能 | `feat/<简短描述>` | `feat/file-rename` |
| 修复 | `fix/<描述>` | `fix/path-traversal-on-windows` |
| 重构 | `refactor/<描述>` | `refactor/extract-sse-writer` |
| 文档 | `docs/<描述>` | `docs/api-spec-update` |
| 杂项 | `chore/<描述>` | `chore/update-go-version` |

### 命名约定

- 使用**短横线**（kebab-case）连接单词
- 描述控制在 2-5 个英文单词
- 小写字母

## 2.5 日常操作流程

```bash
# 1. 确保 main 是最新的
git checkout main
git pull origin main

# 2. 创建功能分支
git checkout -b feat/file-rename

# 3. 开发，提交（使用规范化的 commit message）
git add .
git commit -m "feat(handler): add file rename endpoint"

# 4. 推送
git push origin feat/file-rename

# 5. 在 GitHub 上创建 PR → 通过审查 → 合并
# 6. 删除远程分支
git push origin --delete feat/file-rename
# 删除本地分支
git checkout main && git branch -D feat/file-rename
```

## 2.6 紧急修复流程

对于生产环境的紧急 Bug：

1. 从 main 创建 `fix/xxx` 分支
2. 修复后提交 PR
3. 快速审查后合并
4. 删除分支

---

> **[← 返回索引](./INDEX.md)** · 上一章：[01-总体原则.md](./01-总体原则.md) · 下一章：[03-提交信息规范.md](./03-提交信息规范.md)
