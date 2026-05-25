package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"go-doc-editor/internal/config"
	"go-doc-editor/internal/handler"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"go-doc-editor/internal/user"
)

func main() {
	cfg := config.Load()

	// 用户存储（文件保存在基础存储目录的同级 data 文件夹）
	dataDir := filepath.Join(filepath.Dir(cfg.StorageDir), "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("无法创建数据目录: %v", err)
	}
	userStore, err := user.NewStore(dataDir)
	if err != nil {
		log.Fatalf("初始化用户存储失败: %v", err)
	}

	fileService := service.NewFileService(cfg.StorageDir)
	// 创建 AI 服务
	aiService := service.NewAIService("sk-664c572e17fd40d6aacfa476c05c475e")

	authHandler := handler.NewAuthHandler(userStore)
	fileHandler := handler.NewFileHandler(fileService)
	// 创建 AI 处理器
	aiHandler := handler.NewAIHandler(aiService)

	// 公开路由
	http.HandleFunc("/api/register", authHandler.Register)
	http.HandleFunc("/api/login", authHandler.Login)

	// AI 对话路由（受保护）
	http.HandleFunc("/api/ai/chat", middleware.AuthMiddleware(aiHandler.Chat))

	// 受保护路由（文档操作）
	http.HandleFunc("/api/files", middleware.AuthMiddleware(fileHandler.ListFiles))
	http.HandleFunc("/api/file/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			middleware.AuthMiddleware(fileHandler.ReadFile)(w, r)
		case http.MethodPost:
			middleware.AuthMiddleware(fileHandler.SaveFile)(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(fileHandler.DeleteFile)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 静态文件（前端页面）
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		http.ServeFile(w, r, "./static"+r.URL.Path)
	})

	log.Printf("服务器启动，访问 http://localhost%s", cfg.Port)
	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
