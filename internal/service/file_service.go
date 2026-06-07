// Package service 实现核心业务逻辑层。
// 基于 docstore 提供用户隔离的文档 CRUD 操作。
package service

import (
	"go-doc-editor/internal/docstore"
)

// FileService 提供基于用户隔离的文档 CRUD 操作。
// 内部委托给 docstore.Store 实现数据库存储。
type FileService struct {
	store *docstore.Store
}

// NewFileService 创建 FileService 实例。
// store 是已初始化的文档数据库存储。
func NewFileService(store *docstore.Store) *FileService {
	return &FileService{store: store}
}

// ListFiles 返回当前用户的文件列表，按文件名排序。
func (s *FileService) ListFiles(username string) ([]string, error) {
	return s.store.ListFiles(username)
}

// ReadFile 读取指定文件的内容。
// 文件不存在返回"文件不存在"错误。
func (s *FileService) ReadFile(username, filename string) (string, error) {
	return s.store.ReadFile(username, filename)
}

// SaveFile 保存/覆盖指定文件。
func (s *FileService) SaveFile(username, filename, content string) error {
	return s.store.SaveFile(username, filename, content)
}

// DeleteFile 删除指定文件。
// 文件不存在返回"文件不存在"错误。
func (s *FileService) DeleteFile(username, filename string) error {
	return s.store.DeleteFile(username, filename)
}
