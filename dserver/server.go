package dserver

import (
	"context"
	"log"

	"github.com/colin-404/logx"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/viper"
	"github.com/xid-protocol/common"
)

type Message struct {
	msg     map[string]string
	msgType string
}

// driver server
type DServer struct {
	msgChan   chan Message
	mcpServer *server.MCPServer
}

func InitDServer(debug bool, LogFile string, configFile string, mongoDBURI string, mongoDBDatabase string) *DServer {

	var logLevel int
	if debug {
		logLevel = logx.DebugLevel
	} else {
		logLevel = logx.InfoLevel
	}

	logOpts := &logx.Options{
		Level:   logLevel,
		LogFile: LogFile,
	}
	loger := logx.NewLoger(logOpts)
	logx.InitLogger(loger)

	//load config
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	//init monog
	err = common.InitMongoDB(mongoDBDatabase, mongoDBURI)
	if err != nil {
		logx.Fatalf("Failed to init mongodb: %v", err)
	}
	logx.Infof("config file loaded: %v", viper.ConfigFileUsed())

	ds := &DServer{
		msgChan: make(chan Message),
		mcpServer: server.NewMCPServer("Pentest Workflow Server", "1.0.0",
			server.WithToolCapabilities(true),
			server.WithResourceCapabilities(true, true),
			server.WithRecovery(),
		),
	}

	go ds.handleMessage()

	return ds
}

func (ds *DServer) handleMessage() {
	for msg := range ds.msgChan {
		//send to mcpHost
		if msg.msgType == "log" {
			// Convert map[string]string to map[string]any
			msgMap := make(map[string]any)
			for k, v := range msg.msg {
				msgMap[k] = v
			}
			ds.mcpServer.SendNotificationToClient(context.Background(), "notifications/command", msgMap)
		}
	}
}
