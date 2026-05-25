// config/config.go

package config

import (
	"flag"
	"os"
)

type Config struct {
	Port       string
	StorageDir string
}

func Load() *Config {
	port := flag.String("port", ":3000", "服务器监听端口")
	storageDir := flag.String("storage", "C:\\doxreader", "文档存储目录")

	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = ":" + envPort
	}
	if envDir := os.Getenv("STORAGE_DIR"); envDir != "" {
		*storageDir = envDir
	}

	return &Config{
		Port:       *port,
		StorageDir: *storageDir,
	}
}
