package main

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"apiforge/backend/internal/handler"
	"apiforge/backend/internal/middleware"
	"apiforge/backend/internal/proxy"
	"apiforge/backend/internal/relay"
	grpcproxy "apiforge/backend/internal/grpc"
	"apiforge/backend/internal/server"
	"apiforge/backend/internal/service"
	"apiforge/backend/pkg/config"
	"apiforge/backend/pkg/database"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("加载配置失败", "err", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		slog.Error("打开数据库失败", "err", err)
		os.Exit(1)
	}
	if err := database.Migrate(db); err != nil {
		slog.Error("迁移数据库失败", "err", err)
		os.Exit(1)
	}

	// JWT 密钥兜底：未通过环境变量 APIFORGE_JWT_SECRET 注入时，拒绝使用公开固定串，
	// 改为生成临时随机密钥并告警（重启后失效，生产务必设置环境变量）（H2）。
	if cfg.JWT.Secret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		cfg.JWT.Secret = hex.EncodeToString(b)
		slog.Warn("APIFORGE_JWT_SECRET 未设置，已使用临时随机密钥（重启后失效），生产环境请通过环境变量注入强随机密钥")
	}

	// 注入代理/中继安全配置（SSRF/TLS/超时/消息上限/HTTPS 强制）
	proxy.Configure(cfg.Proxy.AllowPrivateTargets, cfg.Proxy.SkipTLSVerify, cfg.Proxy.DefaultTimeoutMs)
	relay.Configure(cfg.Proxy.AllowPrivateTargets, cfg.Proxy.SkipTLSVerify, 16<<20, cfg.Proxy.RequireHTTPS, cfg.Proxy.MaxConns)

	// 首次启动无用户时创建默认管理员，密码取自 APIFORGE_ADMIN_PASSWORD，
	// 未设置则生成随机密码并告警（不再硬编码弱口令）（H6）
	adminUser := os.Getenv("APIFORGE_ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPwd := os.Getenv("APIFORGE_ADMIN_PASSWORD")
	database.SeedAdmin(db, adminUser, adminPwd)

	// 业务层
	authSvc := service.NewAuthService(db, &cfg.JWT)
	projectSvc := service.NewProjectService(db)
	collectionSvc := service.NewCollectionService(db)
	requestSvc := service.NewRequestService(db)
	environmentSvc := service.NewEnvironmentService(db)
	pipelineSvc := service.NewPipelineService(db, cfg.Proxy.MaxBodyBytes)
	auditSvc := service.NewAuditService(db)

	// 处理器
	authH := handler.NewAuthHandler(authSvc, &cfg.JWT)
	projectH := handler.NewProjectHandler(projectSvc)
	collectionH := handler.NewCollectionHandler(collectionSvc)
	requestH := handler.NewRequestHandler(requestSvc)
	environmentH := handler.NewEnvironmentHandler(environmentSvc)
	proxyH := handler.NewProxyHandler(cfg.Proxy.MaxBodyBytes)
	pipelineH := handler.NewPipelineHandler(pipelineSvc, cfg.Server.PublicURL)
	auditH := handler.NewAuditHandler(auditSvc)

	rt := server.NewRouter()

	// 认证接口免 JWT（前端拿 token 的入口）
	rt.Post("/api/auth/login", http.HandlerFunc(authH.Login))
	rt.Post("/api/auth/register", http.HandlerFunc(authH.Register))
	rt.Post("/api/auth/refresh", http.HandlerFunc(authH.Refresh))
	rt.Post("/api/auth/reset-password", middleware.RequireAuth(http.HandlerFunc(authH.ResetPassword)))
	rt.Post("/api/auth/logout", http.HandlerFunc(authH.Logout))

	// 项目：创建仅需登录（尚不涉及 projectID，故用 RequireAuth，避免 RequireProjectPerm 取空 projectID 必 400）（M1）
	rt.Get("/api/project", http.HandlerFunc(projectH.List))
	rt.Post("/api/project", middleware.RequireAuth(http.HandlerFunc(projectH.Create)))
	rt.Get("/api/project/:projectID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(projectH.Get)))
	rt.Put("/api/project/:projectID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(projectH.Update)))
	rt.Delete("/api/project/:projectID", middleware.RequireProjectPerm(middleware.PermDeleteProject, db)(http.HandlerFunc(projectH.Delete)))
	rt.Get("/api/project/:projectID/members", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(projectH.ListMembers)))
	rt.Get("/api/project/:projectID/members/me", middleware.RequireAuth(http.HandlerFunc(projectH.MyMembership)))
	rt.Post("/api/project/:projectID/members", middleware.RequireProjectPerm(middleware.PermManageMembers, db)(http.HandlerFunc(projectH.AddMember)))
	rt.Put("/api/project/:projectID/members/:userID", middleware.RequireProjectPerm(middleware.PermManageMembers, db)(http.HandlerFunc(projectH.UpdateMember)))
	rt.Delete("/api/project/:projectID/members/:userID", middleware.RequireProjectPerm(middleware.PermManageMembers, db)(http.HandlerFunc(projectH.RemoveMember)))

	// 集合
	rt.Get("/api/project/:projectID/collections", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(collectionH.List)))
	rt.Post("/api/project/:projectID/collections", middleware.RequireProjectPerm(middleware.PermAdd, db)(http.HandlerFunc(collectionH.Create)))
	rt.Put("/api/project/:projectID/collection/:collectionID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(collectionH.Update)))
	rt.Delete("/api/project/:projectID/collection/:collectionID", middleware.RequireProjectPerm(middleware.PermDelete, db)(http.HandlerFunc(collectionH.Delete)))

	// 环境与全局变量（跟项目走，多人共享）
	rt.Get("/api/project/:projectID/environments", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(environmentH.List)))
	rt.Post("/api/project/:projectID/environments", middleware.RequireProjectPerm(middleware.PermAdd, db)(http.HandlerFunc(environmentH.Create)))
	rt.Put("/api/project/:projectID/environment/:envID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(environmentH.Update)))
	rt.Delete("/api/project/:projectID/environment/:envID", middleware.RequireProjectPerm(middleware.PermDelete, db)(http.HandlerFunc(environmentH.Delete)))
	rt.Put("/api/project/:projectID/environment/global", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(environmentH.UpsertGlobal)))
	rt.Post("/api/project/:projectID/environments/reorder", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(environmentH.Reorder)))

	// 保存请求与历史
	rt.Get("/api/project/:projectID/requests", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(requestH.ListAllByProject)))
	rt.Get("/api/project/:projectID/collection/:collectionID/requests", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(requestH.ListByCollection)))
	rt.Post("/api/project/:projectID/collection/:collectionID/requests", middleware.RequireProjectPerm(middleware.PermAdd, db)(http.HandlerFunc(requestH.Save)))
	rt.Get("/api/project/:projectID/request/:requestID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(requestH.Get)))
	rt.Put("/api/project/:projectID/request/:requestID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(requestH.Update)))
	rt.Delete("/api/project/:projectID/request/:requestID", middleware.RequireProjectPerm(middleware.PermDelete, db)(http.HandlerFunc(requestH.Delete)))
	rt.Get("/api/project/:projectID/request/:requestID/history", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(requestH.ListHistory)))
	rt.Post("/api/project/:projectID/request/:requestID/history", middleware.RequireProjectPerm(middleware.PermAdd, db)(http.HandlerFunc(requestH.AddHistory)))

	// HTTP/HTTP2 代理（需登录，免项目权限）
	rt.Post("/api/proxy", proxyH)

	// 审计日志（仅管理员可查看，C9）
	rt.Get("/api/audit", middleware.RequireAdmin(http.HandlerFunc(auditH.List)))

	// 测试流水线（项目权限）
	rt.Get("/api/project/:projectID/pipelines", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(pipelineH.List)))
	rt.Post("/api/project/:projectID/pipelines", middleware.RequireProjectPerm(middleware.PermAdd, db)(http.HandlerFunc(pipelineH.Create)))
	rt.Get("/api/project/:projectID/pipeline/:pipelineID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(pipelineH.Get)))
	rt.Put("/api/project/:projectID/pipeline/:pipelineID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(pipelineH.Update)))
	rt.Delete("/api/project/:projectID/pipeline/:pipelineID", middleware.RequireProjectPerm(middleware.PermDelete, db)(http.HandlerFunc(pipelineH.Delete)))
	rt.Post("/api/project/:projectID/pipeline/:pipelineID/run", middleware.RequireProjectPerm(middleware.PermAdd, db)(http.HandlerFunc(pipelineH.Run)))
	rt.Get("/api/project/:projectID/pipeline/:pipelineID/runs", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(pipelineH.ListRuns)))
	rt.Get("/api/project/:projectID/pipeline/:pipelineID/run/:runID", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(pipelineH.GetRun)))
	rt.Post("/api/project/:projectID/pipeline/:pipelineID/regenerate-token", middleware.RequireProjectPerm(middleware.PermEdit, db)(http.HandlerFunc(pipelineH.RegenerateToken)))

	// Webhook 触发（免鉴权，凭 token 定位流水线）
	rt.Post("/api/webhook/:token", http.HandlerFunc(pipelineH.Webhook))

	// 透传中继：UDP/TCP/MQTT 等协议经后端建立 socket 并双向透传
	relay.Register(relay.TCPHandler{})
	relay.Register(relay.UDPHandler{})
	relay.Register(relay.MQTTHandler{})
	relay.Register(relay.WSHandler{})
	relay.Register(relay.SocketIOHandler{})
	rt.Handle(http.MethodGet, "/ws/relay", http.HandlerFunc(relay.Handler))

	// gRPC 反射代理：前端经 WS 列出服务/方法并调用任意一元 RPC
	rt.Handle(http.MethodGet, "/ws/grpc", http.HandlerFunc(grpcproxy.Handler))

	// 生产期托管前端静态资源，SPA 回退到 index.html
	rt.SetNotFound(server.SpaHandler("./frontend/dist"))

	// 中间件链：CORS 在最外层放行预检；仅 /api(除 auth) 与 /ws 需登录，
	// 静态资源与 SPA 回退对未登录用户开放，保证首屏可加载。
	authed := middleware.JWT(&cfg.JWT)(rt)
	audited := middleware.Audit(cfg.JWT.Secret, auditSvc)(authed)
	root := middleware.CORS(cfg.CORS.AllowOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/auth/") {
			rt.ServeHTTP(w, r)
			return
		}
		// Webhook 触发凭 token 定位流水线，免 JWT 鉴权
		if strings.HasPrefix(p, "/api/webhook/") {
			rt.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/ws") {
			audited.ServeHTTP(w, r)
			return
		}
		rt.ServeHTTP(w, r)
	}))

	addr := ":" + strconv.Itoa(cfg.Server.Port)
	slog.Info("Apiforge 服务启动", "addr", addr)
	// 全局超时，防止慢连接/慢请求无限占用工作线程（M11）
	srv := &http.Server{
		Addr:         addr,
		Handler:      root,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // 代理/流式响应可能长时间写入，交由各 handler 的上下文超时控制
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("服务启动失败", "err", err)
		os.Exit(1)
	}
}
