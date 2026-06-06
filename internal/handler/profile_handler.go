// Package handler 提供 HTTP 处理函数（Controller 层）。
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/user"
)

// ProfileHandler 处理用户个人主页相关的 HTTP 请求。
type ProfileHandler struct {
	userStore *user.Store
}

// NewProfileHandler 创建 ProfileHandler 实例。
func NewProfileHandler(store *user.Store) *ProfileHandler {
	return &ProfileHandler{userStore: store}
}

// profileResponse 是用户主页 API 的响应结构。
type profileResponse struct {
	Success bool         `json:"success"`
	Data    *profileData `json:"data,omitempty"`
	Error   string       `json:"error,omitempty"`
}

type profileData struct {
	Username    string          `json:"username"`
	CreatedAt   string          `json:"createdAt"`
	LastLoginAt string          `json:"lastLoginAt"`
	LoginLogs   []user.LoginLog `json:"loginLogs"`
	AIKeys      []user.AIKey    `json:"aiKeys"`
}

// HandleProfile 返回当前登录用户的个人信息（GET /api/user/profile）。
func (h *ProfileHandler) HandleProfile(w http.ResponseWriter, r *http.Request) {
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"未认证"}`, http.StatusUnauthorized)
		return
	}

	userInfo, err := h.userStore.GetUserInfo(username)
	if err != nil {
		log.Printf("[Profile] 获取用户 %s 信息失败: %v", username, err)
		writeProfileError(w, "获取用户信息失败", http.StatusInternalServerError)
		return
	}

	logs, err := h.userStore.GetLoginLogs(username, 50)
	if err != nil {
		log.Printf("[Profile] 获取用户 %s 登录日志失败: %v", username, err)
		writeProfileError(w, "获取登录日志失败", http.StatusInternalServerError)
		return
	}

	aiKeys, err := h.userStore.GetUserAIKeys(username)
	if err != nil {
		log.Printf("[Profile] 获取用户 %s AI Key 列表失败: %v", username, err)
		// 不阻断，AI Keys 可为空
		aiKeys = []user.AIKey{}
	}

	var lastLoginAt string
	if len(logs) > 0 {
		lastLoginAt = logs[0].LoginTime.Format("2006-01-02 15:04:05")
	}

	resp := profileResponse{
		Success: true,
		Data: &profileData{
			Username:    userInfo.Username,
			CreatedAt:   userInfo.CreatedAt.Format("2006-01-02 15:04:05"),
			LastLoginAt: lastLoginAt,
			LoginLogs:   logs,
			AIKeys:      aiKeys,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ====== AI Key 管理端点 ======

// ListAIKeys 返回当前用户的所有 AI Key（GET /api/user/ai-keys）。
func (h *ProfileHandler) ListAIKeys(w http.ResponseWriter, r *http.Request) {
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		writeProfileError(w, "未认证", http.StatusUnauthorized)
		return
	}
	keys, err := h.userStore.GetUserAIKeys(username)
	if err != nil {
		writeProfileError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "keys": keys})
}

type createAIKeyRequest struct {
	KeyName string `json:"keyName"`
	APIKey  string `json:"apiKey"`
	APIURL  string `json:"apiUrl"`
	Model   string `json:"model"`
}

// CreateAIKey 新增 AI Key（POST /api/user/ai-keys）。
func (h *ProfileHandler) CreateAIKey(w http.ResponseWriter, r *http.Request) {
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		writeProfileError(w, "未认证", http.StatusUnauthorized)
		return
	}
	var req createAIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProfileError(w, "无效请求", http.StatusBadRequest)
		return
	}
	if req.KeyName == "" || req.APIKey == "" {
		writeProfileError(w, "名称和 API Key 不能为空", http.StatusBadRequest)
		return
	}
	// 默认值
	if req.APIURL == "" {
		req.APIURL = "https://api.deepseek.com/chat/completions"
	}
	if req.Model == "" {
		req.Model = "deepseek-v4-flash"
	}
	key, err := h.userStore.CreateAIKey(username, req.KeyName, req.APIKey, req.APIURL, req.Model)
	if err != nil {
		writeProfileError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "key": key})
}

// ActivateAIKey 激活指定 AI Key（PUT /api/user/ai-keys/{id}/activate）。
func (h *ProfileHandler) ActivateAIKey(w http.ResponseWriter, r *http.Request) {
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		writeProfileError(w, "未认证", http.StatusUnauthorized)
		return
	}
	// 从路径中提取 ID: /api/user/ai-keys/{id}/activate
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/user/ai-keys/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		writeProfileError(w, "缺少 Key ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeProfileError(w, "无效的 Key ID", http.StatusBadRequest)
		return
	}
	if err := h.userStore.ActivateAIKey(username, id); err != nil {
		writeProfileError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "已激活"})
}

// DeleteAIKey 删除指定 AI Key（DELETE /api/user/ai-keys/{id}）。
func (h *ProfileHandler) DeleteAIKey(w http.ResponseWriter, r *http.Request) {
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		writeProfileError(w, "未认证", http.StatusUnauthorized)
		return
	}
	// 从路径中提取 ID: /api/user/ai-keys/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/user/ai-keys/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeProfileError(w, "无效的 Key ID", http.StatusBadRequest)
		return
	}
	if err := h.userStore.DeleteAIKey(username, id); err != nil {
		writeProfileError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "已删除"})
}

func writeProfileError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": msg})
}
