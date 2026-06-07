// Package handler 提供标注管理的 HTTP 处理。
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

// AnnotationHandler 处理标注相关的 HTTP 请求。
type AnnotationHandler struct {
	svc *service.FileService
}

// NewAnnotationHandler 创建标注处理器。
func NewAnnotationHandler(svc *service.FileService) *AnnotationHandler {
	return &AnnotationHandler{svc: svc}
}

func (h *AnnotationHandler) writeJSON(w http.ResponseWriter, resp model.Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AnnotationHandler) writeError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(model.Response{Success: false, Error: errMsg})
}

// HandleCreate 创建标注（POST /api/annotations）。
func (h *AnnotationHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	var req model.CreateAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求体", http.StatusBadRequest)
		return
	}
	if req.SourceFilename == "" || req.SelectedText == "" {
		h.writeError(w, "来源文档和选中文本不能为空", http.StatusBadRequest)
		return
	}

	var targetFilename, comment *string
	if req.TargetFilename != "" {
		targetFilename = &req.TargetFilename
	}
	if req.Comment != "" {
		comment = &req.Comment
	}

	id, err := h.svc.CreateAnnotation(username, req.SourceFilename, req.SelectedText, targetFilename, comment, req.PositionStart, req.PositionEnd)
	if err != nil {
		h.writeError(w, "创建标注失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, model.Response{
		Success: true,
		Message: "标注创建成功",
	})
	_ = id
}

// HandleList 获取文档的所有标注（GET /api/file/{filename}/annotations）。
func (h *AnnotationHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	// 路径: /api/file/{filename}/annotations
	path := strings.TrimPrefix(r.URL.Path, "/api/file/")
	filename := strings.TrimSuffix(path, "/annotations")
	if filename == "" {
		h.writeError(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	annos, err := h.svc.GetAnnotations(username, filename)
	if err != nil {
		h.writeError(w, "查询标注失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"annotations": annos,
	})
}

// HandleReferences 获取引用某文档的标注（GET /api/file/{filename}/references）。
func (h *AnnotationHandler) HandleReferences(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	path := strings.TrimPrefix(r.URL.Path, "/api/file/")
	filename := strings.TrimSuffix(path, "/references")
	if filename == "" {
		h.writeError(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	annos, err := h.svc.GetReferences(username, filename)
	if err != nil {
		h.writeError(w, "查询引用失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"references": annos,
	})
}

// HandleDelete 删除标注（DELETE /api/annotations/{id}）。
func (h *AnnotationHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	idStr := strings.TrimPrefix(r.URL.Path, "/api/annotations/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.writeError(w, "无效的标注ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteAnnotation(username, id); err != nil {
		if err.Error() == "标注不存在" {
			h.writeError(w, err.Error(), http.StatusNotFound)
		} else {
			h.writeError(w, "删除失败: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	h.writeJSON(w, model.Response{Success: true, Message: "标注删除成功"})
}
