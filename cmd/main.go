package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cnxysoft/DDBOT-WSa/internal/app"
	"github.com/cnxysoft/DDBOT-WSa/internal/config"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
)

func main() {
	// 初始化日志
	logger.Init()
	defer logger.Sync()

	// 加载配置
	if err := config.Load(); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 创建并启动应用（数据库初始化已内置于 app.Start）
	application := app.New()
	if err := application.Start(); err != nil {
		fmt.Printf("启动应用失败: %v\n", err)
		os.Exit(1)
	}

	// 等待系统退出信号
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	application.Stop()
}
