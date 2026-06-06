// Package model 定义前后端交互的数据契约（DTO）。
// 包含统一的 API 响应结构和请求结构体，所有 handler 层使用这些结构体编码响应。
package model

// Response 是统一 API 响应格式，所有接口返回此结构。
type Response struct {
	Success bool     `json:"success"`           // 操作是否成功
	Message string   `json:"message,omitempty"` // 成功时的提示消息
	Error   string   `json:"error,omitempty"`   // 失败时的错误描述
	Files   []string `json:"files,omitempty"`   // 文件列表（ListFiles 接口使用）
	Content string   `json:"content,omitempty"` // 文件内容（ReadFile 接口使用）
}

// SaveFileRequest 是保存文件接口的请求体。
type SaveFileRequest struct {
	Content string `json:"content"` // 文件内容
}
