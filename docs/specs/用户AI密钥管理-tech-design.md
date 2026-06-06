# 技术方案设计：用户 AI-Key 管理系统

- **日期**: 2026-06-06

---

## 方案对比：AI-Key 存储方式

### 方案 A：独立 `ai_keys` 表（选中 ✅）

新表存储，一对多关系（一个用户可绑定多个 Key）。

**优点**：
- 支持多 Key 槽位
- 支持激活/不激活
- 查询方便
- 后续可扩展更多字段

### 方案 B：在 users 表加字段
- 只能存一个 Key，不支持多槽位
- ❌

### 方案 C：JSON 文件存储
- 失去了数据库优势
- ❌

## 方案对比：AI 调用重构

### 方案 A：ChatStream 接收动态参数（选中 ✅）

```go
func (s *AIService) ChatStream(apiKey, apiURL, model string, messages []DeepseekMsg, onChunk func(string)) (string, error)
```

### 方案 B：AIService 内部查询数据库
- 增加耦合，违反单一职责
- ❌

## 数据库变更

```sql
CREATE TABLE IF NOT EXISTS ai_keys (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    username    VARCHAR(255) NOT NULL,
    key_name    VARCHAR(100) NOT NULL,
    api_key     VARCHAR(255) NOT NULL,
    api_url     VARCHAR(255) NOT NULL DEFAULT 'https://api.deepseek.com/v1/chat/completions',
    model       VARCHAR(100) NOT NULL DEFAULT 'deepseek-chat',
    is_active   TINYINT(1) NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_ai_keys_username (username)
)
```

## 数据流

```
用户打开 Profile 页面
  → GET /api/user/profile 返回用户信息 + AI Key 列表
  → 显示"AI 配置"区域

用户新增 Key：
  POST /api/user/ai-keys {key_name, api_key, api_url, model}
  → 如果这是第一个 Key，自动激活
  → 返回 Key 列表

用户激活 Key：
  PUT /api/user/ai-keys/:id/activate
  → 将该用户的其它 Key 设为 is_active=0
  → 将指定 Key 设为 is_active=1

用户 AI 对话：
  POST /api/ai/chat {messages}
  → handler 从数据库查询用户激活的 Key
  → 如果有激活的 Key：使用用户的 api_key + api_url + model
  → 如果没有激活的 Key：回退全局 cfg.AIKey + 默认 URL + deepseek-chat
  → 调用 ChatStream
```
