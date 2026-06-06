// Package handler 提供 HTTP 处理函数（Controller 层）。
package handler

import (
	"encoding/json"
	"fmt"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"go-doc-editor/internal/user"
	"log"
	"net/http"
)

// AIHandler 处理 AI 对话相关的 HTTP 请求，支持 SSE 流式响应和角色列表查询。
// AI-Key 已改为由用户在个人主页自行配置，不再依赖命令行参数。
type AIHandler struct {
	aiService    *service.AIService
	userStore    *user.Store
	globalAPIKey string // 全局保底 Key（为空时要求用户在 profile 页面配置）
	globalAPIURL string
	globalModel  string
}

// NewAIHandler 创建 AIHandler 实例。
func NewAIHandler(svc *service.AIService, store *user.Store, globalKey, globalURL, globalModel string) *AIHandler {
	return &AIHandler{
		aiService:    svc,
		userStore:    store,
		globalAPIKey: globalKey,
		globalAPIURL: globalURL,
		globalModel:  globalModel,
	}
}

// chatRequest 是 AI 对话接口的请求体，包含前端维护的完整消息历史。
type chatRequest struct {
	Messages []service.DeepseekMsg `json:"messages"`
}

// getAIKeyForUser 获取用户应该使用的 AI 配置。
// 优先级：用户激活的 Key > 全局 Key。如果均不可用返回错误。
func (h *AIHandler) getAIKeyForUser(username string) (apiKey, apiURL, model string, err error) {
	activeKey, dbErr := h.userStore.GetActiveAIKey(username)
	if dbErr != nil {
		log.Printf("[AI] 查询用户 %s 的激活 Key 失败: %v", username, dbErr)
	} else if activeKey != nil {
		return activeKey.APIKey, activeKey.APIURL, activeKey.Model, nil
	}
	// 没有用户 Key，检查全局 Key
	if h.globalAPIKey != "" {
		return h.globalAPIKey, h.globalAPIURL, h.globalModel, nil
	}
	return "", "", "", fmt.Errorf("未配置 AI-Key，请到个人主页添加")
}

// modelDisplayNames 返回模型对应的显示名称。
func modelDisplayName(model string) string {
	switch model {
	case "deepseek-v4-flash", "deepseek-chat":
		return "DeepSeek-v4-Flash（快速模型）"
	case "deepseek-v4-pro", "deepseek-reasoner":
		return "DeepSeek-v4-Pro（推理模型）"
	default:
		return model
	}
}

// injectModelIdentity 在消息列表中注入模型身份信息。
// 如果是 system 消息，在其内容后追加模型身份；否则新增一条 system 消息。
func injectModelIdentity(messages []service.DeepseekMsg, model string) []service.DeepseekMsg {
	displayName := modelDisplayName(model)
	identityLine := "\n\n【你的身份】你当前的模型是 " + displayName + "。如果用户问起你是什么模型，请如实告知。"

	result := make([]service.DeepseekMsg, 0, len(messages)+1)
	for _, msg := range messages {
		if msg.Role == "system" {
			msg.Content = msg.Content + identityLine
		}
		result = append(result, msg)
	}
	// 如果没有 system 消息，在最前面插入一条
	if len(messages) == 0 || messages[0].Role != "system" {
		result = append([]service.DeepseekMsg{{
			Role:    "system",
			Content: "【你的身份】你当前的模型是 " + displayName + "。如果用户问起你是什么模型，请如实告知。",
		}}, result...)
	}
	return result
}

// Chat 处理 AI 对话请求（POST /api/ai/chat），以 SSE 格式流式返回 AI 响应。
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"未认证"}`, http.StatusUnauthorized)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求体"}`, http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, `{"error":"消息列表不能为空"}`, http.StatusBadRequest)
		return
	}

	apiKey, apiURL, model, keyErr := h.getAIKeyForUser(username)
	if keyErr != nil {
		log.Printf("[AI] 用户 %s 无可用 AI-Key: %v", username, keyErr)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, keyErr.Error()), http.StatusBadRequest)
		return
	}
	log.Printf("[AI] 用户 %s 发起 AI 请求，模型: %s，消息数量: %d", username, model, len(req.Messages))

	// 注入模型身份消息，让 AI 知道自己在用什么模型
	messagesWithIdentity := injectModelIdentity(req.Messages, model)

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"不支持流式传输"}`, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", `{"type":"start"}`)
	flusher.Flush()

	fullContent, err := h.aiService.ChatStream(apiKey, apiURL, model, messagesWithIdentity, func(text string) {
		escaped := escapeSSE(text)
		fmt.Fprintf(w, "data: %s\n\n", escaped)
		flusher.Flush()
	})

	if err != nil {
		log.Printf("[AI] 用户 %s AI 请求失败: %v", username, err)
		errJSON, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", string(errJSON))
		flusher.Flush()
		return
	}

	doneJSON, _ := json.Marshal(map[string]string{
		"type":    "done",
		"content": fullContent,
	})
	fmt.Fprintf(w, "data: %s\n\n", string(doneJSON))
	flusher.Flush()

	log.Printf("[AI] 用户 %s AI 请求完成，总长度: %d", username, len(fullContent))
}

// ListRoles 返回预设角色列表（GET /api/ai/roles）。
func (h *AIHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"roles":   service.GetRoles(),
	})
}

// escapeSSE 将文本内容包装为 SSE 格式的 JSON 数据块。
func escapeSSE(text string) string {
	escaped, _ := json.Marshal(map[string]string{"type": "chunk", "delta": text})
	return string(escaped)
}