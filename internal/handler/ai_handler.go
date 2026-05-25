package handler

import (
	"encoding/json"
	"fmt"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"log"
	"net/http"
)

type AIHandler struct {
	aiService *service.AIService
}

func NewAIHandler(svc *service.AIService) *AIHandler {
	return &AIHandler{aiService: svc}
}

type chatRequest struct {
	Messages []service.DeepseekMsg `json:"messages"` // 完整的对话历史（前端维护）
}

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

// SSE 转义：将内容包装为 SSE data 格式
func escapeSSE(text string) string {
	// 每块只包含纯文本 delta
	escaped, _ := json.Marshal(map[string]string{"type": "chunk", "delta": text})
	return string(escaped)
}
