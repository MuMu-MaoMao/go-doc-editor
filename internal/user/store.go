// Package user 实现基于 MySQL 的用户存储和管理。
// 密码使用 bcrypt 加密存储，支持注册时间记录和登录日志追踪。
package user

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User 表示一个用户账户，包含注册时间。
type User struct {
	Username  string    `json:"username"`
	Password  string    `json:"-"` // 不序列化到 JSON
	CreatedAt time.Time `json:"createdAt"`
}

// LoginLog 表示一次登录记录。
type LoginLog struct {
	LoginTime time.Time `json:"loginTime"`
}

// Store 提供用户存储和登录日志的查询操作，基于 MySQL。
type Store struct {
	db *sql.DB
}

// NewStore 创建基于 MySQL 的用户存储。
func NewStore(database *sql.DB) *Store {
	return &Store{db: database}
}

// CreateUser 注册新用户，对密码进行 bcrypt 加密后存储。
// 如果用户名已存在，返回 ErrUserExists。
func (s *Store) CreateUser(username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}
	_, err = s.db.Exec(
		"INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, NOW())",
		username, string(hash),
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return ErrUserExists
		}
		return fmt.Errorf("创建用户失败: %v", err)
	}
	return nil
}

// ValidateUser 验证用户名和密码是否匹配。
// 如果用户不存在返回 ErrUserNotFound，密码不匹配返回 bcrypt 错误。
func (s *Store) ValidateUser(username, password string) error {
	var hash string
	err := s.db.QueryRow(
		"SELECT password_hash FROM users WHERE username = ?", username,
	).Scan(&hash)
	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("查询用户失败: %v", err)
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GetUserInfo 获取用户信息（注册时间）。
func (s *Store) GetUserInfo(username string) (*User, error) {
	user := &User{Username: username}
	err := s.db.QueryRow(
		"SELECT created_at FROM users WHERE username = ?", username,
	).Scan(&user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户信息失败: %v", err)
	}
	return user, nil
}

// RecordLogin 记录一次登录日志。
func (s *Store) RecordLogin(username string) error {
	_, err := s.db.Exec(
		"INSERT INTO login_logs (username, login_time) VALUES (?, NOW())",
		username,
	)
	if err != nil {
		return fmt.Errorf("记录登录日志失败: %v", err)
	}
	return nil
}

// GetLoginLogs 获取用户的登录历史，按时间倒序排列。
func (s *Store) GetLoginLogs(username string, limit int) ([]LoginLog, error) {
	rows, err := s.db.Query(
		"SELECT login_time FROM login_logs WHERE username = ? ORDER BY login_time DESC LIMIT ?",
		username, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("查询登录日志失败: %v", err)
	}
	defer rows.Close()

	var logs []LoginLog
	for rows.Next() {
		var log LoginLog
		if err := rows.Scan(&log.LoginTime); err != nil {
			return nil, fmt.Errorf("解析登录日志失败: %v", err)
		}
		logs = append(logs, log)
	}
	if logs == nil {
		logs = []LoginLog{} // 返回空数组而非 null
	}
	return logs, nil
}

// isDuplicateEntry 检查 MySQL 错误是否为重复键（用户已存在）。
func isDuplicateEntry(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062"))
}

// 预定义的错误变量，供 handler 层判断错误类型。
var (
	ErrUserExists   = &userError{"用户已存在"}
	ErrUserNotFound = &userError{"用户名或密码错误"}
)

type userError struct{ msg string }

func (e *userError) Error() string { return e.msg }
