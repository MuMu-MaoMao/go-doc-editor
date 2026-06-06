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

// AIKey 表示一个用户绑定的 AI API 密钥配置。
type AIKey struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	KeyName   string    `json:"keyName"`
	APIKey    string    `json:"apiKey,omitempty"` // 新增时传入，查询时隐藏
	APIURL    string    `json:"apiUrl"`
	Model     string    `json:"model"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetUserAIKeys 获取用户的所有 AI Key 配置。
func (s *Store) GetUserAIKeys(username string) ([]AIKey, error) {
	rows, err := s.db.Query(
		`SELECT id, username, key_name, api_url, model, is_active, created_at
		 FROM ai_keys WHERE username = ? ORDER BY created_at DESC`, username,
	)
	if err != nil {
		return nil, fmt.Errorf("查询 AI Key 列表失败: %v", err)
	}
	defer rows.Close()

	var keys []AIKey
	for rows.Next() {
		var k AIKey
		if err := rows.Scan(&k.ID, &k.Username, &k.KeyName, &k.APIURL, &k.Model, &k.IsActive, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("解析 AI Key 失败: %v", err)
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []AIKey{}
	}
	return keys, nil
}

// CreateAIKey 为用户新增一个 AI Key 配置。
// 如果这是用户的第一个 Key，自动设为激活。
func (s *Store) CreateAIKey(username, keyName, apiKey, apiURL, model string) (*AIKey, error) {
	// 判断是否已有 Key
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM ai_keys WHERE username = ?", username).Scan(&count)
	isActive := count == 0 // 第一个 Key 自动激活

	result, err := s.db.Exec(
		`INSERT INTO ai_keys (username, key_name, api_key, api_url, model, is_active)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		username, keyName, apiKey, apiURL, model, isActive,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 AI Key 失败: %v", err)
	}
	id, _ := result.LastInsertId()
	return &AIKey{ID: id, Username: username, KeyName: keyName, APIURL: apiURL, Model: model, IsActive: isActive}, nil
}

// ActivateAIKey 激活指定 AI Key，同时取消该用户其他 Key 的激活状态。
func (s *Store) ActivateAIKey(username string, keyID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("事务开始失败: %v", err)
	}
	defer tx.Rollback()

	// 取消所有激活
	if _, err := tx.Exec("UPDATE ai_keys SET is_active = 0 WHERE username = ?", username); err != nil {
		return fmt.Errorf("取消激活失败: %v", err)
	}
	// 激活指定 Key
	result, err := tx.Exec("UPDATE ai_keys SET is_active = 1 WHERE id = ? AND username = ?", keyID, username)
	if err != nil {
		return fmt.Errorf("激活失败: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("未找到该 Key")
	}
	return tx.Commit()
}

// DeleteAIKey 删除用户的指定 AI Key。
func (s *Store) DeleteAIKey(username string, keyID int64) error {
	result, err := s.db.Exec("DELETE FROM ai_keys WHERE id = ? AND username = ?", keyID, username)
	if err != nil {
		return fmt.Errorf("删除 AI Key 失败: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("未找到该 Key")
	}
	return nil
}

// GetActiveAIKey 获取用户当前激活的 AI Key（含 API Key 明文）。
func (s *Store) GetActiveAIKey(username string) (*AIKey, error) {
	k := &AIKey{Username: username}
	err := s.db.QueryRow(
		`SELECT id, key_name, api_key, api_url, model, is_active, created_at
		 FROM ai_keys WHERE username = ? AND is_active = 1 LIMIT 1`, username,
	).Scan(&k.ID, &k.KeyName, &k.APIKey, &k.APIURL, &k.Model, &k.IsActive, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // 没有激活的 Key
	}
	if err != nil {
		return nil, fmt.Errorf("查询激活 Key 失败: %v", err)
	}
	return k, nil
}

// 预定义的错误变量，供 handler 层判断错误类型。
var (
	ErrUserExists   = &userError{"用户已存在"}
	ErrUserNotFound = &userError{"用户名或密码错误"}
)

type userError struct{ msg string }

func (e *userError) Error() string { return e.msg }
