# Spec：用户 AI-Key 管理系统

- **版本**: v1.0
- **日期**: 2026-06-06

---

## Objective

让用户自主管理 AI-Key，无需命令行参数，支持多 Key 槽位、模型选择。

## Tech Stack

同项目现有技术栈（Go + MySQL + 原生前端），新增依赖：无。

## Project Structure

```
internal/
├── db/mysql.go                    # [修改] InitTables 增加 ai_keys 表
├── user/store.go                  # [修改] 新增 AIKey CRUD 方法
├── service/ai_service.go          # [修改] ChatStream 改为动态参数
├── handler/ai_handler.go          # [修改] Chat 从数据库获取用户 Key
└── handler/profile_handler.go     # [修改] profile 返回 AI Key 列表 + 管理端点

static/
├── profile.html                   # [修改] 新增 AI 配置 UI
└── profile.js                     # [修改] AI Key 管理逻辑
```

## API 接口

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | `/api/user/ai-keys` | JWT | 获取当前用户所有 AI Key |
| POST | `/api/user/ai-keys` | JWT | 新增一个 AI Key |
| PUT | `/api/user/ai-keys/:id/activate` | JWT | 激活指定 Key |
| DELETE | `/api/user/ai-keys/:id` | JWT | 删除指定 Key |

## Boundaries

**Always**: SQL 参数化查询、Key 不暴露在日志中、用户只能操作自己的 Key
**Never**: Key 不做加密（当前阶段信任本地部署）

## Success Criteria

同需求文档接受标准。
