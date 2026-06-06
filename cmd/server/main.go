package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"go-doc-editor/internal/config"
	"go-doc-editor/internal/db"
	"go-doc-editor/internal/handler"
	"go-doc-editor/internal/middleware"
	"go-doc-editor/internal/service"
	"go-doc-editor/internal/user"
)

func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

func main() {
	cfg := config.Load()

	if cfg.AIKey == "" || cfg.AIKey == "your-ai-key-here" {
		log.Fatal("请通过 --ai-key 参数提供 AI 服务的 API Key（config.json 中是占位符，需要替换）")
	}

	// 检查 MySQL 连接串
	if cfg.MySQLDSN == "" {
		log.Fatal("请通过 --mysql-dsn 参数提供 MySQL 连接串，" +
			"例如: --mysql-dsn \"root:密码@tcp(localhost:3306)/godoxedit?charset=utf8mb4&parseTime=true\"")
	}

	// 连接 MySQL
	database, err := db.NewDB(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer database.Close()

	// 初始化表结构
	if err := db.InitTables(database); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Println("数据库连接成功，表结构已就绪")

	// 获取项目根目录
	projectRoot := getProjectRoot()
	staticDir := filepath.Join(projectRoot, "static")

	// 数据目录（用于文件存储，不再用于用户 JSON）
	if err := os.MkdirAll(filepath.Join(filepath.Dir(cfg.StorageDir), "data"), 0755); err != nil {
		log.Fatalf("无法创建数据目录: %v", err)
	}

	// 各层初始化
	userStore := user.NewStore(database)
	fileService := service.NewFileService(cfg.StorageDir)
	aiService := service.NewAIService(cfg.AIKey)

	authHandler := handler.NewAuthHandler(userStore)
	fileHandler := handler.NewFileHandler(fileService)
	aiHandler := handler.NewAIHandler(aiService)
	profileHandler := handler.NewProfileHandler(userStore)

	// 公开路由
	http.HandleFunc("/api/register", authHandler.Register)
	http.HandleFunc("/api/login", authHandler.Login)
	http.HandleFunc("/api/ai/roles", aiHandler.ListRoles)

	// 受保护路由
	http.HandleFunc("/api/ai/chat", middleware.AuthMiddleware(aiHandler.Chat))
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
	http.HandleFunc("/api/user/profile", middleware.AuthMiddleware(profileHandler.HandleProfile))

	// 静态文件
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
