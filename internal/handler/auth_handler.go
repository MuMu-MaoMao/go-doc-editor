package handler

import (
	"encoding/json"
	"net/http"

	"go-doc-editor/internal/auth"
	"go-doc-editor/internal/user"
)

type AuthHandler struct {
	userStore *user.Store
}

func NewAuthHandler(store *user.Store) *AuthHandler {
	return &AuthHandler{userStore: store}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, "无效请求", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		writeAuthError(w, "用户名和密码不能为空", http.StatusBadRequest)
		return
	}
	if err := h.userStore.CreateUser(req.Username, req.Password); err != nil {
		if err == user.ErrUserExists {
			writeAuthError(w, err.Error(), http.StatusConflict)
		} else {
			writeAuthError(w, "注册失败", http.StatusInternalServerError)
		}
		return
	}
	writeAuthSuccess(w, "注册成功，请登录")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, "无效请求", http.StatusBadRequest)
		return
	}
	if err := h.userStore.ValidateUser(req.Username, req.Password); err != nil {
		writeAuthError(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}
	token, err := auth.GenerateToken(req.Username)
	if err != nil {
		writeAuthError(w, "生成令牌失败", http.StatusInternalServerError)
		return
	}
	resp := authResponse{Success: true, Token: token, Message: "登录成功"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeAuthError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(authResponse{Success: false, Error: msg})
}

func writeAuthSuccess(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse{Success: true, Message: msg})
}
