package handler

import (
	"encoding/json"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/model"
	"go-doc-editor/internal/service"
	"net/http"
	"strings"
)

type FileHandler struct {
	service *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{service: svc}
}

func (h *FileHandler) writeJSON(w http.ResponseWriter, resp model.Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *FileHandler) writeError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(model.Response{Success: false, Error: errMsg})
}

func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())
	files, err := h.service.ListFiles(username)
	if err != nil {
		h.writeError(w, "读取目录失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, model.Response{Success: true, Files: files})
}

func (h *FileHandler) ReadFile(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/file/")
	if filename == "" {
		h.writeError(w, "文件名不能为空", http.StatusBadRequest)
		return
	}
	username, _ := middleware.GetUsernameFromContext(r.Context())
	content, err := h.service.ReadFile(username, filename)
	if err != nil {
		if err.Error() == "文件不存在" {
			h.writeError(w, err.Error(), http.StatusNotFound)
		} else {
			h.writeError(w, "读取文件失败: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	h.writeJSON(w, model.Response{Success: true, Content: content})
}

func (h *FileHandler) SaveFile(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/file/")
	if filename == "" {
		h.writeError(w, "文件名不能为空", http.StatusBadRequest)
		return
	}
	var req model.SaveFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求体", http.StatusBadRequest)
		return
	}
	username, _ := middleware.GetUsernameFromContext(r.Context())
	if err := h.service.SaveFile(username, filename, req.Content); err != nil {
		h.writeError(w, "保存失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, model.Response{Success: true, Message: "文件保存成功"})
}

func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/file/")
	if filename == "" {
		h.writeError(w, "文件名不能为空", http.StatusBadRequest)
		return
	}
	username, _ := middleware.GetUsernameFromContext(r.Context())
	if err := h.service.DeleteFile(username, filename); err != nil {
		if err.Error() == "文件不存在" {
			h.writeError(w, err.Error(), http.StatusNotFound)
		} else {
			h.writeError(w, "删除失败: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	h.writeJSON(w, model.Response{Success: true, Message: "文件删除成功"})
}
