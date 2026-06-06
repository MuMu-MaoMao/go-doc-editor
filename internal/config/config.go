// Package config 加载和管理项目的运行配置。
// 支持配置文件、命令行参数和环境变量三种配置方式，优先级：
// 环境变量 > 命令行参数 > 配置文件
package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Config 存储项目的运行参数。
type Config struct {
	Port       string `json:"port"`       // 服务监听端口，如 ":3000"
	StorageDir string `json:"storage"`       // 文档存储根目录，如 "C:\\doxreader"
	MySQLDSN   string `json:"mysql_dsn"`     // MySQL 连接串
	AIKey      string `json:"ai_key"`        // DeepSeek API 密钥
}

// findProjectRoot 从当前源文件位置推算项目根目录。
// config.go 在 internal/config/，向上两级即项目根目录。
func findProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filename))
}

// loadConfigFile 读取项目根目录的 config.json，不存在时返回空配置。
func loadConfigFile() *Config {
	cfgPath := filepath.Join(findProjectRoot(), "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return &Config{}
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("警告: config.json 解析失败: %v，使用默认值", err)
		return &Config{}
	}
	return &cfg
}

// notEmpty 返回 s 如果非空，否则返回 fallback。
func notEmpty(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}

// Load 加载配置，优先级：环境变量 > 命令行参数 > config.json。
func Load() *Config {
	// 1. 读取 config.json 作为默认值
	fileCfg := loadConfigFile()

	// 2. 注册命令行参数（以 config.json 的值为默认值）
	port := flag.String("port", notEmpty(fileCfg.Port, ":3000"), "服务器监听端口")
	storageDir := flag.String("storage", notEmpty(fileCfg.StorageDir, "C:\\doxreader"), "文档存储目录")
	mysqlDSN := flag.String("mysql-dsn", fileCfg.MySQLDSN, "MySQL 连接串")
	aiKey := flag.String("ai-key", fileCfg.AIKey, "API Key for AI service")

	flag.Parse()

	// 3. 环境变量覆盖
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = ":" + envPort
	}
	if envDir := os.Getenv("STORAGE_DIR"); envDir != "" {
		*storageDir = envDir
	}
	if envDSN := os.Getenv("MYSQL_DSN"); envDSN != "" {
		*mysqlDSN = envDSN
	}
	if envKey := os.Getenv("AI_KEY"); envKey != "" {
		*aiKey = envKey
	}

	return &Config{
		Port:       *port,
		StorageDir: *storageDir,
		MySQLDSN:   *mysqlDSN,
		AIKey:      *aiKey,
	}
}
