package main

import (
	"context"
	"log"
	"time"

	"github.com/xid-protocol/mcp-driver/examples/multi_task/server/biz"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/colin-404/logx"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xid-protocol/mcp-driver/connection/ssepool"
)

var (
	logFile string
	debug   bool
)

var rootCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manager",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func setupConfig() {

	viper.Set("debug", debug)
	viper.Set("logfile", logFile)
	if debug {
		log.Printf("config: %v", viper.AllSettings())
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug level")
	rootCmd.PersistentFlags().StringVarP(&logFile, "logFile", "l", "./logs/log.log", "log file path")
}

func startServer() {
	ctx := context.Background()
	//初始化sse连接池
	pool := ssepool.New(1024)
	//启动清理协程
	go pool.StartCleanup(1*time.Hour, 3*time.Hour)

	//事件驱动
	// pubSub := gochannel.NewGoChannel(
	// 	gochannel.Config{},
	// 	watermill.NewStdLogger(false, false),
	// )

	// messages, err := pubSub.Subscribe(ctx, "example.topic")
	// if err != nil {
	// 	logx.Errorf("Failed to subscribe to topic: %v", err)
	// }

	// go process(messages)

	// publishMessages(pubSub)
	react.NewAgent(ctx, nil)

	//gin配置
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	biz.RegisterRouter(router)

	logx.Infof("Listening and serving on %s", "0.0.0.0:8080")
	err := router.Run("0.0.0.0:8080")
	if err != nil {
		logx.Errorf("SRV_ERROR %s", err.Error())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logx.Fatalf("Error executing command: %v", err)
	}
}
