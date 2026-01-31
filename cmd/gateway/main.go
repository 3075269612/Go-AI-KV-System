package main

import (
	"Go-AI-KV-System/internal/gateway/handler"
	"Go-AI-KV-System/internal/gateway/router"
	"Go-AI-KV-System/pkg/client"
	"Go-AI-KV-System/pkg/logger"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	// 1. 初始化配置
	viper.SetDefault("server.mode", "debug")        // 默认开发模式
	viper.SetDefault("server.port", "8080")         // 默认端口
	viper.SetDefault("rpc.addr", "127.0.0.1:50051") // gRPC 服务端地址配置 (使用 IPv4 避免 localhost 解析延迟)

	// 2. 初始化日志
	logger.InitLogger()
	// 程序退出前刷新日志缓冲区，防止日志丢失
	defer logger.Log.Sync()

	// 获取全局 Logger 实例
	log := logger.Log
	log.Info("🚀 Gateway is starting...")

	// 3. 设置 Gin 的运行模式
	gin.SetMode(viper.GetString("server.mode"))

	// 新增：gRPC Client 连接逻辑
	rpcAddr := viper.GetString("rpc.addr")
	log.Info("🔗 Connecting to gRPC Server...", zap.String("addr", rpcAddr))

	// 初始化 gRPC 客户端
	kvClient, err := client.NewClient(rpcAddr)
	if err != nil {
		log.Fatal("❌ Failed to connect to KV Server", zap.Error(err))
	}
	defer func() {
		log.Info("🔌 Closing gRPC connection...")
		if err := kvClient.Close(); err != nil {
			log.Error("Failed to close gRPC connection", zap.Error(err))
		}
	}()

	// 4. 初始化 Handlers (控制层)
	kvHandler := handler.NewKVHandler(kvClient)
	healthHandler := handler.NewHealthHandler()

	// 5. 初始化 Router (路由层)
	r := router.NewRouter(kvHandler, healthHandler)

	// 6. 配置 HTTP Server
	port := viper.GetString("server.port")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// 7. 启动服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("❌ Listen error", zap.Error(err))
		}
	}()
	log.Info("✅ Gateway running", zap.String("port", port))

	// 8. 优雅退出
	quit := make(chan os.Signal, 1)
	// 监听中断信号 (Ctrl+C, Docker stop)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞直到收到信号
	<-quit
	log.Info("⚠️ Shutting down gateway...")

	// 创建一个 5 秒超时的 Context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭服务器，处理完当前的请求
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ Server forced to shutdown", zap.Error(err))
	}

	log.Info("👋 Gateway exited properly")
}
