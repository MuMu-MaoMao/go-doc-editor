// Package handler 提供 HTTP 处理函数（Controller 层）。
package handler

import (
	"encoding/json"
	"log"
	"net/http"

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
	Success    bool            `json:"success"`
	Data       *profileData    `json:"data,omitempty"`
	Error      string          `json:"error,omitempty"`
}

type profileData struct {
	Username     string          `json:"username"`
	CreatedAt    string          `json:"createdAt"`
	LastLoginAt  string          `json:"lastLoginAt"`
	LoginLogs    []user.LoginLog `json:"loginLogs"`
}

// HandleProfile 返回当前登录用户的个人信息（GET /api/user/profile）。
func (h *ProfileHandler) HandleProfile(w http.ResponseWriter, r *http.Request) {
	username, ok := middleware.GetUsernameFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"未认证"}`, http.StatusUnauthorized)
		return
	}

	// 获取用户信息
	userInfo, err := h.userStore.GetUserInfo(username)
	if err != nil {
		log.Printf("[Profile] 获取用户 %s 信息失败: %v", username, err)
		writeProfileError(w, "获取用户信息失败", http.StatusInternalServerError)
		return
	}

	// 获取登录日志（最近 50 条）
	logs, err := h.userStore.GetLoginLogs(username, 50)
	if err != nil {
		log.Printf("[Profile] 获取用户 %s 登录日志失败: %v", username, err)
		writeProfileError(w, "获取登录日志失败", http.StatusInternalServerError)
		return
	}

	// 计算最近登录时间
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
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeProfileError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(profileResponse{Success: false, Error: msg})
}
