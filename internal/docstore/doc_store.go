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

// ErrFileNotFound 表示请求的文档不存在。
var ErrFileNotFound = &storeError{"文件不存在"}

type storeError struct{ msg string }

func (e *storeError) Error() string { return e.msg }
