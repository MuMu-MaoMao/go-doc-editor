// Package docstore 提供基于 MySQL 的文档 CRUD 存储。
// 所有操作基于用户名进行用户隔离，通过参数化查询防止 SQL 注入。
// 与 user.Store 采用相同的架构模式。
package docstore

import (
	"database/sql"
	"fmt"
)

// Store 提供文档的数据库 CRUD 操作。
// 每个方法接收 username 参数确保用户隔离。
type Store struct {
	db *sql.DB
}

// NewStore 创建基于 MySQL 的文档存储。
// db 是已连接的 *sql.DB 实例。
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ListFiles 返回用户的所有文档文件名列表，按文件名 ASC 排序。
func (s *Store) ListFiles(username string) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT filename FROM documents WHERE username = ? ORDER BY filename ASC",
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("查询文档列表失败: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("解析文档名失败: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历文档列表失败: %v", err)
	}
	if names == nil {
		names = []string{} // 返回空数组而非 null
	}
	return names, nil
}

// ReadFile 读取用户指定文件的内容。
// 文件不存在返回 ErrFileNotFound。
func (s *Store) ReadFile(username, filename string) (string, error) {
	var content string
	err := s.db.QueryRow(
		"SELECT content FROM documents WHERE username = ? AND filename = ?",
		username, filename,
	).Scan(&content)
	if err == sql.ErrNoRows {
		return "", ErrFileNotFound
	}
	if err != nil {
		return "", fmt.Errorf("读取文档失败: %v", err)
	}
	return content, nil
}

// SaveFile 保存/覆盖用户指定文件。
// 使用 INSERT … ON DUPLICATE KEY UPDATE 实现 upsert 语义：
//   - 如果文件已存在，更新内容和 updated_at
//   - 如果文件不存在，创建新记录
func (s *Store) SaveFile(username, filename, content string) error {
	_, err := s.db.Exec(
		`INSERT INTO documents (username, filename, content, created_at, updated_at)
		 VALUES (?, ?, ?, NOW(), NOW())
		 ON DUPLICATE KEY UPDATE content = VALUES(content), updated_at = NOW()`,
		username, filename, content,
	)
	if err != nil {
		return fmt.Errorf("保存文档失败: %v", err)
	}
	return nil
}

// DeleteFile 删除用户指定文件。
// 文件不存在返回 ErrFileNotFound。
func (s *Store) DeleteFile(username, filename string) error {
	result, err := s.db.Exec(
		"DELETE FROM documents WHERE username = ? AND filename = ?",
		username, filename,
	)
	if err != nil {
		return fmt.Errorf("删除文档失败: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrFileNotFound
	}
	return nil
}

// ============================================================
// 分类（Category）CRUD
// ============================================================

// Category 表示数据库中的分类行。
type Category struct {
	ID        int64
	Name      string
	ParentID  *int64
	SortOrder int
}

// CreateCategory 创建一个新分类，返回生成的 ID。
// parentID 为 nil 表示创建顶级大类。
// 同级下不允许同名分类。
func (s *Store) CreateCategory(username, name string, parentID *int64) (int64, error) {
	// 先检查同级下是否已有同名分类
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM categories WHERE username = ? AND parent_id <=> ? AND name = ?",
		username, parentID, name,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("检查分类重名失败: %v", err)
	}
	if count > 0 {
		return 0, ErrCategoryNameDuplicate
	}

	result, err := s.db.Exec(
		`INSERT INTO categories (username, name, parent_id, sort_order)
		 VALUES (?, ?, ?, COALESCE((SELECT MAX(sort_order)+1 FROM categories c2 WHERE c2.username = ? AND c2.parent_id <=> ?), 0))`,
		username, name, parentID, username, parentID,
	)
	if err != nil {
		return 0, fmt.Errorf("创建分类失败: %v", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// GetAllCategories 获取用户的所有分类（扁平列表，由应用层组装为树）。
func (s *Store) GetAllCategories(username string) ([]Category, error) {
	rows, err := s.db.Query(
		"SELECT id, name, parent_id, sort_order FROM categories WHERE username = ? ORDER BY sort_order ASC",
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("查询分类失败: %v", err)
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &parentID, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("解析分类失败: %v", err)
		}
		if parentID.Valid {
			c.ParentID = &parentID.Int64
		}
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分类失败: %v", err)
	}
	if cats == nil {
		cats = []Category{}
	}
	return cats, nil
}

// GetCategory 获取单个分类信息。不存在返回 ErrCategoryNotFound。
func (s *Store) GetCategory(username string, categoryID int64) (*Category, error) {
	var c Category
	var parentID sql.NullInt64
	err := s.db.QueryRow(
		"SELECT id, name, parent_id, sort_order FROM categories WHERE id = ? AND username = ?",
		categoryID, username,
	).Scan(&c.ID, &c.Name, &parentID, &c.SortOrder)
	if err == sql.ErrNoRows {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询分类失败: %v", err)
	}
	if parentID.Valid {
		c.ParentID = &parentID.Int64
	}
	return &c, nil
}

// RenameCategory 修改分类名称。
func (s *Store) RenameCategory(username string, categoryID int64, newName string) error {
	result, err := s.db.Exec(
		"UPDATE categories SET name = ? WHERE id = ? AND username = ?",
		newName, categoryID, username,
	)
	if err != nil {
		return fmt.Errorf("重命名分类失败: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

// DeleteCategory 删除分类。分类下有子分类或文档时返回相应的错误。
func (s *Store) DeleteCategory(username string, categoryID int64) error {
	// 检查是否有子分类
	var childCount int
	s.db.QueryRow("SELECT COUNT(*) FROM categories WHERE parent_id = ? AND username = ?", categoryID, username).Scan(&childCount)
	if childCount > 0 {
		return ErrCategoryHasChildren
	}

	// 检查是否有文档属于此分类
	var docCount int
	s.db.QueryRow("SELECT COUNT(*) FROM documents WHERE category_id = ? AND username = ?", categoryID, username).Scan(&docCount)
	if docCount > 0 {
		return ErrCategoryHasDocs
	}

	result, err := s.db.Exec("DELETE FROM categories WHERE id = ? AND username = ?", categoryID, username)
	if err != nil {
		return fmt.Errorf("删除分类失败: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

// ============================================================
// 文档-分类关联
// ============================================================

// SetDocumentCategory 设置文档的分类。categoryID 为 nil 表示取消分类。
func (s *Store) SetDocumentCategory(username, filename string, categoryID *int64) error {
	if categoryID != nil {
		// 验证分类存在
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = ? AND username = ?", *categoryID, username).Scan(&count)
		if count == 0 {
			return ErrCategoryNotFound
		}
	}
	_, err := s.db.Exec(
		"UPDATE documents SET category_id = ? WHERE username = ? AND filename = ?",
		categoryID, username, filename,
	)
	if err != nil {
		return fmt.Errorf("设置文档分类失败: %v", err)
	}
	return nil
}

// GetDocumentCategory 获取某文档所属的分类ID。
func (s *Store) GetDocumentCategory(username, filename string) (*int64, error) {
	var categoryID sql.NullInt64
	err := s.db.QueryRow(
		"SELECT category_id FROM documents WHERE username = ? AND filename = ?",
		username, filename,
	).Scan(&categoryID)
	if err == sql.ErrNoRows {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询文档分类失败: %v", err)
	}
	if categoryID.Valid {
		return &categoryID.Int64, nil
	}
	return nil, nil
}

// ListFilesByCategory 返回某分类下的所有文档文件名列表。
func (s *Store) ListFilesByCategory(username string, categoryID int64) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT filename FROM documents WHERE username = ? AND category_id = ? ORDER BY filename ASC",
		username, categoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询分类文档失败: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("解析文档名失败: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分类文档失败: %v", err)
	}
	if names == nil {
		names = []string{}
	}
	return names, nil
}

// ListUncategorizedFiles 返回未分类的文档文件名列表。
func (s *Store) ListUncategorizedFiles(username string) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT filename FROM documents WHERE username = ? AND category_id IS NULL ORDER BY filename ASC",
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("查询未分类文档失败: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("解析文档名失败: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历未分类文档失败: %v", err)
	}
	if names == nil {
		names = []string{}
	}
	return names, nil
}

// ============================================================
// 标注（Annotation）CRUD
// ============================================================

// Annotation 表示数据库中的一行标注。
type Annotation struct {
	ID             int64
	SourceFilename string
	SelectedText   string
	TargetFilename *string
	Comment        *string
	PositionStart  int
	PositionEnd    int
}

// CreateAnnotation 创建一个标注，返回生成的 ID。
func (s *Store) CreateAnnotation(username, sourceFilename, selectedText string, targetFilename, comment *string, positionStart, positionEnd int) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO annotations (username, source_filename, selected_text, target_filename, comment, position_start, position_end)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		username, sourceFilename, selectedText, targetFilename, comment, positionStart, positionEnd,
	)
	if err != nil {
		return 0, fmt.Errorf("创建标注失败: %v", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// GetAnnotations 获取某文档的所有标注，按位置排序。
func (s *Store) GetAnnotations(username, sourceFilename string) ([]Annotation, error) {
	rows, err := s.db.Query(
		"SELECT id, source_filename, selected_text, target_filename, comment, position_start, position_end FROM annotations WHERE username = ? AND source_filename = ? ORDER BY position_start ASC",
		username, sourceFilename,
	)
	if err != nil {
		return nil, fmt.Errorf("查询标注失败: %v", err)
	}
	defer rows.Close()

	var annotations []Annotation
	for rows.Next() {
		var a Annotation
		var targetFilename, comment sql.NullString
		if err := rows.Scan(&a.ID, &a.SourceFilename, &a.SelectedText, &targetFilename, &comment, &a.PositionStart, &a.PositionEnd); err != nil {
			return nil, fmt.Errorf("解析标注失败: %v", err)
		}
		if targetFilename.Valid {
			a.TargetFilename = &targetFilename.String
		}
		if comment.Valid {
			a.Comment = &comment.String
		}
		annotations = append(annotations, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历标注失败: %v", err)
	}
	if annotations == nil {
		annotations = []Annotation{}
	}
	return annotations, nil
}

// GetReferences 获取引用某文档的所有标注（"被引用于"）。
func (s *Store) GetReferences(username, targetFilename string) ([]Annotation, error) {
	rows, err := s.db.Query(
		"SELECT id, source_filename, selected_text, target_filename, comment, position_start, position_end FROM annotations WHERE username = ? AND target_filename = ? ORDER BY created_at DESC",
		username, targetFilename,
	)
	if err != nil {
		return nil, fmt.Errorf("查询引用失败: %v", err)
	}
	defer rows.Close()

	var annotations []Annotation
	for rows.Next() {
		var a Annotation
		var targetFilename, comment sql.NullString
		if err := rows.Scan(&a.ID, &a.SourceFilename, &a.SelectedText, &targetFilename, &comment, &a.PositionStart, &a.PositionEnd); err != nil {
			return nil, fmt.Errorf("解析引用失败: %v", err)
		}
		if targetFilename.Valid {
			a.TargetFilename = &targetFilename.String
		}
		if comment.Valid {
			a.Comment = &comment.String
		}
		annotations = append(annotations, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历引用失败: %v", err)
	}
	if annotations == nil {
		annotations = []Annotation{}
	}
	return annotations, nil
}

// DeleteAnnotation 删除标注。不存在返回 ErrAnnotationNotFound。
func (s *Store) DeleteAnnotation(username string, annotationID int64) error {
	result, err := s.db.Exec("DELETE FROM annotations WHERE id = ? AND username = ?", annotationID, username)
	if err != nil {
		return fmt.Errorf("删除标注失败: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAnnotationNotFound
	}
	return nil
}

// ErrFileNotFound 表示请求的文档不存在。
var ErrFileNotFound = &storeError{"文件不存在"}

// ErrCategoryNotFound 表示请求的分类不存在。
var ErrCategoryNotFound = &storeError{"分类不存在"}

// ErrCategoryHasChildren 表示分类下还有子分类，无法删除。
var ErrCategoryHasChildren = &storeError{"分类下还有子分类"}

// ErrCategoryHasDocs 表示分类下还有文档，无法删除。
var ErrCategoryHasDocs = &storeError{"分类下还有文档"}

// ErrCategoryNameDuplicate 表示同级下已存在同名分类。
var ErrCategoryNameDuplicate = &storeError{"同级下已存在同名分类"}

// ErrAnnotationNotFound 表示请求的标注不存在。
var ErrAnnotationNotFound = &storeError{"标注不存在"}

type storeError struct{ msg string }

func (e *storeError) Error() string { return e.msg }
