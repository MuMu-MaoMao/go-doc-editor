package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"go-doc-editor/internal/config"
	"go-doc-editor/internal/handler"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"go-doc-editor/internal/user"
)

func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// main.go 在 cmd/server/main.go，向上三级到项目根目录
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

func main() {
	// 定义命令行参数 --ai-key
	aiKey := flag.String("ai-key", "", "API Key for AI service (required)")
	flag.Parse()

	if *aiKey == "" {
		log.Fatal("请通过 --ai-key 参数提供 AI 服务的 API Key，例如: go run main.go --ai-key=你的新Key")
	}

	cfg := config.Load()

	// 获取项目根目录（基于源文件位置，不依赖工作目录）
	projectRoot := getProjectRoot()
	staticDir := filepath.Join(projectRoot, "static")

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
	// 使用命令行参数传入的 Key 创建 AI 服务
	aiService := service.NewAIService(*aiKey)

	authHandler := handler.NewAuthHandler(userStore)
	fileHandler := handler.NewFileHandler(fileService)
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
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		http.ServeFile(w, r, filepath.Join(staticDir, r.URL.Path))
	})

	log.Printf("服务器启动，访问 http://localhost%s", cfg.Port)
	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
