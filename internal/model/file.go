// model/file.go

package model

type Response struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
	Files   []string `json:"files,omitempty"`
	Content string   `json:"content,omitempty"`
}

type SaveFileRequest struct {
	Content string `json:"content"`
}
