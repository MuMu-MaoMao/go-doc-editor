package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileService struct {
	baseDir string // 基础存储目录，如 C:\doxreader
}

func NewFileService(baseDir string) *FileService {
	return &FileService{baseDir: baseDir}
}

// 获取用户专属目录
func (s *FileService) getUserDir(username string) string {
	return filepath.Join(s.baseDir, username)
}

// 确保用户目录存在
func (s *FileService) EnsureUserDir(username string) error {
	userDir := s.getUserDir(username)
	return os.MkdirAll(userDir, 0755)
}

// 安全路径校验，基于具体用户目录
func (s *FileService) safeFilePath(username, filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("文件名不能为空")
	}
	if strings.ContainsAny(filename, "/\\") || strings.Contains(filename, "..") {
		return "", fmt.Errorf("文件名包含非法字符")
	}
	userDir := s.getUserDir(username)
	absDir, err := filepath.Abs(userDir)
	if err != nil {
		return "", fmt.Errorf("无法解析目录: %v", err)
	}
	absPath, err := filepath.Abs(filepath.Join(userDir, filename))
	if err != nil {
		return "", fmt.Errorf("无法解析路径: %v", err)
	}
	if !strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) && absPath != absDir {
		return "", fmt.Errorf("非法路径访问")
	}
	return absPath, nil
}

func (s *FileService) ListFiles(username string) ([]string, error) {
	if err := s.EnsureUserDir(username); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.getUserDir(username))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (s *FileService) ReadFile(username, filename string) (string, error) {
	safePath, err := s.safeFilePath(username, filename)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在")
	}
	content, err := os.ReadFile(safePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (s *FileService) SaveFile(username, filename, content string) error {
	safePath, err := s.safeFilePath(username, filename)
	if err != nil {
		return err
	}
	if err := s.EnsureUserDir(username); err != nil {
		return err
	}
	return os.WriteFile(safePath, []byte(content), 0644)
}

func (s *FileService) DeleteFile(username, filename string) error {
	safePath, err := s.safeFilePath(username, filename)
	if err != nil {
		return err
	}
	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在")
	}
	return os.Remove(safePath)
}
