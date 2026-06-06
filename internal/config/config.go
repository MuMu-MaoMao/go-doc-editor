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
	StorageDir string `json:"storage"`    // 文档存储根目录，如 "C:\\doxreader"
	MySQLDSN   string `json:"mysql_dsn"`  // MySQL 连接串
}

// findConfigFile 尝试多个路径查找 config.json，返回第一个找到的。
func findConfigFile() string {
	_, srcFile, _, _ := runtime.Caller(0)
	candidates := []string{
		filepath.Join(filepath.Dir(filepath.Dir(srcFile)), "config.json"),
		filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(srcFile))), "config.json"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "config.json"))
		candidates = append(candidates, filepath.Join(wd, "..", "config.json"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// loadConfigFile 读取项目根目录的 config.json，不存在时返回空配置。
func loadConfigFile() *Config {
	cfgPath := findConfigFile()
	if cfgPath == "" {
		return &Config{}
	}
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
// 数据库和 AI-Key 等敏感信息由用户在 profile 页面自行配置。
func Load() *Config {
	fileCfg := loadConfigFile()

	port := flag.String("port", notEmpty(fileCfg.Port, ":3000"), "服务器监听端口")
	storageDir := flag.String("storage", notEmpty(fileCfg.StorageDir, "C:\\doxreader"), "文档存储目录")
	mysqlDSN := flag.String("mysql-dsn", fileCfg.MySQLDSN, "MySQL 连接串")

	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = ":" + envPort
	}
	if envDir := os.Getenv("STORAGE_DIR"); envDir != "" {
		*storageDir = envDir
	}
	if envDSN := os.Getenv("MYSQL_DSN"); envDSN != "" {
		*mysqlDSN = envDSN
	}

	return &Config{
		Port:       *port,
		StorageDir: *storageDir,
		MySQLDSN:   *mysqlDSN,
	}
}
