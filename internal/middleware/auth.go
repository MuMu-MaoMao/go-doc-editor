// Package middleware 提供 HTTP 中间件，用于请求的前置处理（如认证拦截）。
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go-doc-editor/internal/auth"
)

// contextKey 用于在 context 中存储请求范围的值，避免键冲突。
type contextKey string

// UserContextKey 是 context 中存储当前用户名的键。
// 通过 AuthMiddleware 验证成功后注入，handler 通过 GetUsernameFromContext 获取。
const UserContextKey contextKey = "username"

// AuthMiddleware 验证 HTTP 请求中的 JWT Bearer Token。
// 验证成功后，将用户名注入请求 context；失败时返回 401 JSON 响应。
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "未提供认证令牌",
			})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "令牌格式错误",
			})
			return
		}
		claims, err := auth.ValidateToken(parts[1])
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "无效或过期的令牌",
			})
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUsernameFromContext 从请求 context 中提取用户名。
// 返回用户名和是否存在的布尔值。通常在 AuthMiddleware 之后调用。
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UserContextKey).(string)
	return username, ok
}
