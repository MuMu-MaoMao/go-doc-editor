// Package service 实现核心业务逻辑层。
// 基于 docstore 提供用户隔离的文档 CRUD 操作和分类管理。
package service

import (
	"go-doc-editor/internal/docstore"
	"go-doc-editor/internal/model"
)

// FileService 提供基于用户隔离的文档 CRUD 操作和分类管理。
// 内部委托给 docstore.Store 实现数据库存储。
type FileService struct {
	store *docstore.Store
}

// NewFileService 创建 FileService 实例。
// store 是已初始化的文档数据库存储。
func NewFileService(store *docstore.Store) *FileService {
	return &FileService{store: store}
}

// ---- 文档 CRUD ----

// ListFiles 返回当前用户的文件列表，按文件名排序。
func (s *FileService) ListFiles(username string) ([]string, error) {
	return s.store.ListFiles(username)
}

// ReadFile 读取指定文件的内容。
func (s *FileService) ReadFile(username, filename string) (string, error) {
	return s.store.ReadFile(username, filename)
}

// SaveFile 保存/覆盖指定文件。
func (s *FileService) SaveFile(username, filename, content string) error {
	return s.store.SaveFile(username, filename, content)
}

// DeleteFile 删除指定文件。
func (s *FileService) DeleteFile(username, filename string) error {
	return s.store.DeleteFile(username, filename)
}

// ---- 分类管理 ----

// docCategoryToModel 将 docstore.Category 转为 model.Category。
func docCategoryToModel(c docstore.Category) model.Category {
	return model.Category{
		ID:        c.ID,
		Name:      c.Name,
		ParentID:  c.ParentID,
		SortOrder: c.SortOrder,
	}
}

// CreateCategory 创建一个新分类，返回生成的 ID。
func (s *FileService) CreateCategory(username, name string, parentID *int64) (int64, error) {
	return s.store.CreateCategory(username, name, parentID)
}

// GetAllCategories 获取用户的所有分类（扁平列表）。
func (s *FileService) GetAllCategories(username string) ([]model.Category, error) {
	cats, err := s.store.GetAllCategories(username)
	if err != nil {
		return nil, err
	}
	result := make([]model.Category, len(cats))
	for i, c := range cats {
		result[i] = docCategoryToModel(c)
	}
	return result, nil
}

// GetCategory 获取单个分类信息。
func (s *FileService) GetCategory(username string, categoryID int64) (*model.Category, error) {
	c, err := s.store.GetCategory(username, categoryID)
	if err != nil {
		return nil, err
	}
	mc := docCategoryToModel(*c)
	return &mc, nil
}

// RenameCategory 修改分类名称。
func (s *FileService) RenameCategory(username string, categoryID int64, newName string) error {
	return s.store.RenameCategory(username, categoryID, newName)
}

// DeleteCategory 删除分类。
func (s *FileService) DeleteCategory(username string, categoryID int64) error {
	return s.store.DeleteCategory(username, categoryID)
}

// SetDocumentCategory 设置文档的分类。
func (s *FileService) SetDocumentCategory(username, filename string, categoryID *int64) error {
	return s.store.SetDocumentCategory(username, filename, categoryID)
}

// GetDocumentCategory 获取某文档的分类ID。
func (s *FileService) GetDocumentCategory(username, filename string) (*int64, error) {
	return s.store.GetDocumentCategory(username, filename)
}

// ListFilesByCategory 返回某分类下的所有文档。
func (s *FileService) ListFilesByCategory(username string, categoryID int64) ([]string, error) {
	return s.store.ListFilesByCategory(username, categoryID)
}

// ListUncategorizedFiles 返回未分类的文档。
func (s *FileService) ListUncategorizedFiles(username string) ([]string, error) {
	return s.store.ListUncategorizedFiles(username)
}

// ---- 标注管理 ----

// CreateAnnotation 创建一个标注。
func (s *FileService) CreateAnnotation(username, sourceFilename, selectedText string, targetFilename, comment *string, positionStart, positionEnd int) (int64, error) {
	return s.store.CreateAnnotation(username, sourceFilename, selectedText, targetFilename, comment, positionStart, positionEnd)
}

// GetAnnotations 获取某文档的所有标注。
func (s *FileService) GetAnnotations(username, sourceFilename string) ([]model.Annotation, error) {
	annos, err := s.store.GetAnnotations(username, sourceFilename)
	if err != nil {
		return nil, err
	}
	result := make([]model.Annotation, len(annos))
	for i, a := range annos {
		m := model.Annotation{
			ID:             a.ID,
			SourceFilename: a.SourceFilename,
			SelectedText:   a.SelectedText,
			PositionStart:  a.PositionStart,
			PositionEnd:    a.PositionEnd,
		}
		if a.TargetFilename != nil {
			m.TargetFilename = *a.TargetFilename
		}
		if a.Comment != nil {
			m.Comment = *a.Comment
		}
		result[i] = m
	}
	return result, nil
}

// GetReferences 获取引用某文档的标注。
func (s *FileService) GetReferences(username, targetFilename string) ([]model.Annotation, error) {
	annos, err := s.store.GetReferences(username, targetFilename)
	if err != nil {
		return nil, err
	}
	result := make([]model.Annotation, len(annos))
	for i, a := range annos {
		m := model.Annotation{
			ID:             a.ID,
			SourceFilename: a.SourceFilename,
			SelectedText:   a.SelectedText,
			PositionStart:  a.PositionStart,
			PositionEnd:    a.PositionEnd,
		}
		if a.TargetFilename != nil {
			m.TargetFilename = *a.TargetFilename
		}
		if a.Comment != nil {
			m.Comment = *a.Comment
		}
		result[i] = m
	}
	return result, nil
}

// DeleteAnnotation 删除标注。
func (s *FileService) DeleteAnnotation(username string, annotationID int64) error {
	return s.store.DeleteAnnotation(username, annotationID)
}

// Category 类型别名，供 handler 使用
type Category = model.Category
