// Package db 提供 MySQL 数据库连接管理和表初始化功能。
package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// NewDB 建立 MySQL 数据库连接，dsn 格式：
// user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true
func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("无法打开数据库连接: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("无法连接到数据库: %v", err)
	}
	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}

// InitTables 创建项目所需的数据库表（不存在则创建）。
func InitTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			username      VARCHAR(255) PRIMARY KEY,
			password_hash VARCHAR(255) NOT NULL,
			created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS login_logs (
			id         BIGINT       AUTO_INCREMENT PRIMARY KEY,
			username   VARCHAR(255) NOT NULL,
			login_time DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_login_logs_username (username),
			INDEX idx_login_logs_time (login_time)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS documents (
			id         BIGINT       AUTO_INCREMENT PRIMARY KEY,
			username   VARCHAR(255) NOT NULL,
			filename   VARCHAR(255) NOT NULL,
			content    LONGTEXT,
			created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE INDEX idx_documents_user_file (username, filename),
			INDEX idx_documents_username (username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS ai_keys (
			id         BIGINT       AUTO_INCREMENT PRIMARY KEY,
			username   VARCHAR(255) NOT NULL,
			key_name   VARCHAR(100) NOT NULL,
			api_key    VARCHAR(255) NOT NULL,
			api_url    VARCHAR(255) NOT NULL DEFAULT 'https://api.deepseek.com/chat/completions',
			model      VARCHAR(100) NOT NULL DEFAULT 'deepseek-v4-flash',
			is_active  TINYINT(1)  NOT NULL DEFAULT 0,
			created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_ai_keys_username (username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("建表失败: %v", err)
		}
	}

	// 新增表：分类树
	extraTables := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id         BIGINT       AUTO_INCREMENT PRIMARY KEY,
			username   VARCHAR(255) NOT NULL,
			name       VARCHAR(100) NOT NULL,
			parent_id  BIGINT       DEFAULT NULL,
			sort_order INT          NOT NULL DEFAULT 0,
			created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_cat_username (username),
			INDEX idx_cat_parent (username, parent_id),
			UNIQUE INDEX idx_cat_name_unique (username, parent_id, name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS annotations (
			id               BIGINT       AUTO_INCREMENT PRIMARY KEY,
			username         VARCHAR(255) NOT NULL,
			source_filename  VARCHAR(255) NOT NULL,
			selected_text    TEXT         NOT NULL,
			target_filename  VARCHAR(255) DEFAULT NULL,
			comment          TEXT,
			position_start   INT          NOT NULL DEFAULT 0,
			position_end     INT          NOT NULL DEFAULT 0,
			created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_anno_source (username, source_filename),
			INDEX idx_anno_target (username, target_filename)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, q := range extraTables {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("建表失败: %v", err)
		}
	}

	// documents 表加 category_id 字段（已存在则跳过）
	if _, err := db.Exec(
		`ALTER TABLE documents
		 ADD COLUMN category_id BIGINT DEFAULT NULL COMMENT '所属分类ID',
		 ADD INDEX idx_doc_category (username, category_id)`,
	); err != nil {
		// 字段已存在时忽略 Duplicate column 错误
		if !strings.Contains(err.Error(), "Duplicate column") {
			return fmt.Errorf("更新 documents 表失败: %v", err)
		}
	}

	return nil
}
