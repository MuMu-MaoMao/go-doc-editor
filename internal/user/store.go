package user

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"` // bcrypt hash
}

type Store struct {
	filePath string
	mu       sync.RWMutex
	users    map[string]User
}

func NewStore(dataDir string) (*Store, error) {
	store := &Store{
		filePath: filepath.Join(dataDir, "users.json"),
		users:    make(map[string]User),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.users)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) CreateUser(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[username]; exists {
		return ErrUserExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.users[username] = User{Username: username, Password: string(hash)}
	return s.save()
}

func (s *Store) ValidateUser(username, password string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[username]
	if !exists {
		return ErrUserNotFound
	}
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

var (
	ErrUserExists   = &userError{"用户已存在"}
	ErrUserNotFound = &userError{"用户名或密码错误"}
)

type userError struct{ msg string }

func (e *userError) Error() string { return e.msg }
