// Package handler 提供分类管理的 HTTP 处理。
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

// CategoryHandler 处理分类相关的 HTTP 请求。
type CategoryHandler struct {
	service *service.FileService
}

// NewCategoryHandler 创建分类处理器。
func NewCategoryHandler(svc *service.FileService) *CategoryHandler {
	return &CategoryHandler{service: svc}
}

func (h *CategoryHandler) writeJSON(w http.ResponseWriter, resp model.Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *CategoryHandler) writeError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(model.Response{Success: false, Error: errMsg})
}

// parseCategoryID 从 URL 路径中提取分类 ID。/api/categories/{id} → id
func parseCategoryID(path, prefix string) (int64, error) {
	idStr := strings.TrimPrefix(path, prefix)
	idStr = strings.TrimSuffix(idStr, "/")
	return strconv.ParseInt(idStr, 10, 64)
}

// buildCategoryTree 将扁平分类列表组装为树形结构。
func buildCategoryTree(cats []model.Category) []*model.Category {
	catMap := make(map[int64]*model.Category)
	var roots []*model.Category

	// 第一轮：创建节点
	for _, c := range cats {
		node := &model.Category{
			ID:        c.ID,
			Name:      c.Name,
			ParentID:  c.ParentID,
			SortOrder: c.SortOrder,
		}
		catMap[c.ID] = node
	}

	// 第二轮：挂载到父节点
	for _, node := range catMap {
		if node.ParentID == nil {
			roots = append(roots, node)
		} else {
			parent, ok := catMap[*node.ParentID]
			if ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	if roots == nil {
		roots = []*model.Category{}
	}
	return roots
}

// HandleListTree 返回用户的完整分类树（GET /api/categories）。
func (h *CategoryHandler) HandleListTree(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())
	cats, err := h.service.GetAllCategories(username)
	if err != nil {
		h.writeError(w, "获取分类失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tree := buildCategoryTree(cats)
	h.writeJSON(w, model.Response{Success: true, Tree: tree})
}

// HandleCreate 创建分类（POST /api/categories）。
func (h *CategoryHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	var req model.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求体", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		h.writeError(w, "分类名称不能为空", http.StatusBadRequest)
		return
	}

	// 校验三级深度限制
	if req.ParentID != nil {
		parent, err := h.service.GetCategory(username, *req.ParentID)
		if err != nil {
			h.writeError(w, "父分类不存在", http.StatusBadRequest)
			return
		}
		// 计算父分类的层级：按 parent_id 链向上追溯
		depth := 1
		currentParentID := parent.ParentID
		for currentParentID != nil {
			depth++
			if depth >= 3 {
				h.writeError(w, "最多支持三级分类（大类→中类→小类）", http.StatusBadRequest)
				return
			}
			grandparent, err := h.service.GetCategory(username, *currentParentID)
			if err != nil {
				break
			}
			currentParentID = grandparent.ParentID
		}
	}

	id, err := h.service.CreateCategory(username, req.Name, req.ParentID)
	if err != nil {
		h.writeError(w, "创建分类失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	category := &model.Category{ID: id, Name: req.Name, ParentID: req.ParentID}
	h.writeJSON(w, model.Response{Success: true, Message: "分类创建成功", Category: category})
}

// HandleRename 重命名分类（PUT /api/categories/{id}）。
func (h *CategoryHandler) HandleRename(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	categoryID, err := parseCategoryID(r.URL.Path, "/api/categories/")
	if err != nil {
		h.writeError(w, "无效的分类ID", http.StatusBadRequest)
		return
	}

	var req model.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求体", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		h.writeError(w, "分类名称不能为空", http.StatusBadRequest)
		return
	}

	if err := h.service.RenameCategory(username, categoryID, req.Name); err != nil {
		if err.Error() == "分类不存在" {
			h.writeError(w, err.Error(), http.StatusNotFound)
		} else {
			h.writeError(w, "重命名失败: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	h.writeJSON(w, model.Response{Success: true, Message: "重命名成功"})
}

// HandleDelete 删除分类（DELETE /api/categories/{id}）。
func (h *CategoryHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	categoryID, err := parseCategoryID(r.URL.Path, "/api/categories/")
	if err != nil {
		h.writeError(w, "无效的分类ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteCategory(username, categoryID); err != nil {
		msg := err.Error()
		if msg == "分类不存在" {
			h.writeError(w, msg, http.StatusNotFound)
		} else {
			h.writeError(w, msg, http.StatusBadRequest)
		}
		return
	}
	h.writeJSON(w, model.Response{Success: true, Message: "分类删除成功"})
}

// HandleSetDocCategory 设置文档的分类（PUT /api/file/{filename}/category）。
func (h *CategoryHandler) HandleSetDocCategory(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	// 路径格式: /api/file/{filename}/category
	path := strings.TrimPrefix(r.URL.Path, "/api/file/")
	filename := strings.TrimSuffix(path, "/category")
	if filename == "" {
		h.writeError(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	var req model.SetCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	if err := h.service.SetDocumentCategory(username, filename, req.CategoryID); err != nil {
		if err.Error() == "分类不存在" || err.Error() == "文件不存在" {
			h.writeError(w, err.Error(), http.StatusNotFound)
		} else {
			h.writeError(w, "设置分类失败: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	h.writeJSON(w, model.Response{Success: true, Message: "分类设置成功"})
}

// HandleGetDocCategory 获取文档的分类（GET /api/file/{filename}/category）。
func (h *CategoryHandler) HandleGetDocCategory(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.GetUsernameFromContext(r.Context())

	path := strings.TrimPrefix(r.URL.Path, "/api/file/")
	filename := strings.TrimSuffix(path, "/category")
	if filename == "" {
		h.writeError(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	categoryID, err := h.service.GetDocumentCategory(username, filename)
	if err != nil {
		if err.Error() == "文件不存在" {
			h.writeError(w, err.Error(), http.StatusNotFound)
		} else {
			h.writeError(w, "查询失败: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// 如果有分类，返回分类信息
	if categoryID != nil {
		cat, err := h.service.GetCategory(username, *categoryID)
		if err == nil {
			mcat := &model.Category{ID: cat.ID, Name: cat.Name, ParentID: cat.ParentID, SortOrder: cat.SortOrder}
			h.writeJSON(w, model.Response{Success: true, Category: mcat})
			return
		}
	}
	h.writeJSON(w, model.Response{Success: true})
}
