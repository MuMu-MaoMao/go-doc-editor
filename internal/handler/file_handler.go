package handler

import (
	"encoding/json"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/model"
	"go-doc-editor/internal/service"
	"net/http"
	"strconv"
	"strings"
)

// FileHandler 处理文档 CRUD 相关的 HTTP 请求。
type FileHandler struct {
	service *service.FileService
}

// NewFileHandler 创建 FileHandler 实例，依赖文件服务。
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

// ListFiles 返回当前用户的文件列表（GET /api/files）。
// 支持查询参数：
//
//	?category=ID     只返回指定分类下的文件
//	?uncategorized=1  只返回未分类的文件
func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	categoryParam := r.URL.Query().Get("category")
	uncategorizedParam := r.URL.Query().Get("uncategorized")

	var files []string
	var err error

	if uncategorizedParam == "1" {
		files, err = h.service.ListUncategorizedFiles(username)
	} else if categoryParam != "" {
		catID, parseErr := strconv.ParseInt(categoryParam, 10, 64)
		if parseErr != nil {
			h.writeError(w, "无效的分类ID", http.StatusBadRequest)
			return
		}
		files, err = h.service.ListFilesByCategory(username, catID)
	} else {
		files, err = h.service.ListFiles(username)
	}

	if err != nil {
		h.writeError(w, "读取文件列表失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, model.Response{Success: true, Files: files})
}

// ReadFile 读取指定文件的内容（GET /api/file/{filename}）。
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

// SaveFile 保存/覆盖指定文件（POST /api/file/{filename}）。
// 请求体 JSON：{"content": "文件内容"}。
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

// DeleteFile 删除指定文件（DELETE /api/file/{filename}）。
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
