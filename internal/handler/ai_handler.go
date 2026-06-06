// Package handler 提供 HTTP 处理函数（Controller 层）。
// 负责解析 HTTP 请求、参数校验、调用 Service 层、编码响应。
package handler

import (
	"encoding/json"
	"fmt"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"log"
	"net/http"
)

// AIHandler 处理 AI 对话相关的 HTTP 请求，支持 SSE 流式响应和角色列表查询。
type AIHandler struct {
	aiService *service.AIService
}

// NewAIHandler 创建 AIHandler 实例，依赖 AI 服务。
func NewAIHandler(svc *service.AIService) *AIHandler {
	return &AIHandler{aiService: svc}
}

// chatRequest 是 AI 对话接口的请求体，包含前端维护的完整消息历史。
type chatRequest struct {
	Messages []service.DeepseekMsg `json:"messages"`
}

// Chat 处理 AI 对话请求（POST /api/ai/chat），以 SSE 格式流式返回 AI 响应。
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	// 验证登录（中间件已处理）
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

	log.Printf("[AI] 用户 %s 发起 AI 请求，消息数量: %d", username, len(req.Messages))

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

	// 发送开始标记
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"start"}`)
	flusher.Flush()

	// 调用 AI 服务，流式返回
	fullContent, err := h.aiService.ChatStream(req.Messages, func(text string) {
		// 对内容中的换行和特殊字符进行转义，确保 SSE 格式正确
		escaped := escapeSSE(text)
		fmt.Fprintf(w, "data: %s\n\n", escaped)
		flusher.Flush()
	})

	if err != nil {
		log.Printf("[AI] 用户 %s AI 请求失败: %v", username, err)
		// 发送错误事件
		errJSON, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", string(errJSON))
		flusher.Flush()
		return
	}

	// 发送完成标记（附带完整内容用于前端记录历史）
	doneJSON, _ := json.Marshal(map[string]string{
		"type":    "done",
		"content": fullContent,
	})
	fmt.Fprintf(w, "data: %s\n\n", string(doneJSON))
	flusher.Flush()

	log.Printf("[AI] 用户 %s AI 请求完成，总长度: %d", username, len(fullContent))
}

// ListRoles 返回预设角色列表（GET /api/ai/roles）。
// 此接口无需认证，返回角色 ID、名称和描述（不含 systemPrompt）。
func (h *AIHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"roles":   service.GetRoles(),
	})
}

// escapeSSE 将文本内容包装为 SSE 格式的 JSON 数据块。
func escapeSSE(text string) string {
	// 每块只包含纯文本 delta
	escaped, _ := json.Marshal(map[string]string{"type": "chunk", "delta": text})
	return string(escaped)
}
