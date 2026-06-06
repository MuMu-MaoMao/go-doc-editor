package main

import (
    "log"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
    "strings"

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

    if cfg.MySQLDSN == "" {
        log.Fatal("请通过 --mysql-dsn 参数提供 MySQL 连接串")
    }

    database, err := db.NewDB(cfg.MySQLDSN)
    if err != nil {
        log.Fatalf("数据库连接失败: %v", err)
    }
    defer database.Close()

    if err := db.InitTables(database); err != nil {
        log.Fatalf("数据库初始化失败: %v", err)
    }
    log.Println("数据库连接成功，表结构已就绪")

    projectRoot := getProjectRoot()
    staticDir := filepath.Join(projectRoot, "static")

    if err := os.MkdirAll(filepath.Join(filepath.Dir(cfg.StorageDir), "data"), 0755); err != nil {
        log.Fatalf("无法创建数据目录: %v", err)
    }

    userStore := user.NewStore(database)
    fileService := service.NewFileService(cfg.StorageDir)
    aiService := service.NewAIService()

    authHandler := handler.NewAuthHandler(userStore)
    fileHandler := handler.NewFileHandler(fileService)
    aiHandler := handler.NewAIHandler(aiService, userStore, "", "https://api.deepseek.com/chat/completions", "deepseek-v4-flash")
    profileHandler := handler.NewProfileHandler(userStore)

    http.HandleFunc("/api/register", authHandler.Register)
    http.HandleFunc("/api/login", authHandler.Login)
    http.HandleFunc("/api/ai/roles", aiHandler.ListRoles)

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

    http.HandleFunc("/api/user/ai-keys", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            middleware.AuthMiddleware(profileHandler.ListAIKeys)(w, r)
        case http.MethodPost:
            middleware.AuthMiddleware(profileHandler.CreateAIKey)(w, r)
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })
    http.HandleFunc("/api/user/ai-keys/", func(w http.ResponseWriter, r *http.Request) {
        if strings.HasSuffix(r.URL.Path, "/activate") {
            if r.Method == http.MethodPut {
                middleware.AuthMiddleware(profileHandler.ActivateAIKey)(w, r)
            } else {
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            }
            return
        }
        if r.Method == http.MethodDelete {
            middleware.AuthMiddleware(profileHandler.DeleteAIKey)(w, r)
        } else {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })

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
