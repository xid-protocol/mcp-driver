package dserver

import (
	"context"
	"log"
	"net/http"

	"github.com/colin-404/logx"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/xid-protocol/common"
)

// type Message struct {
// 	msg     map[string]string
// 	msgType string
// }

// driver server
type DServer struct {
	EventChan chan map[string]any
	MCPServer *server.MCPServer
}

type DBType string

const (
	DBTypeMongoDB DBType = "mongodb"
)

type DBInfo struct {
	Type     DBType
	URI      string
	Database string
}

func InitDServer(debug bool, LogFile string, dbInfo DBInfo) *DServer {

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

	//init monogo
	if dbInfo.Type == DBTypeMongoDB {
		err := common.InitMongoDB(dbInfo.Database, dbInfo.URI, true)
		if err != nil {
			logx.Fatalf("Failed to init mongodb: %v", err)
		}
	}

	//init dserver
	ds := &DServer{
		EventChan: make(chan map[string]any, 100),
		MCPServer: server.NewMCPServer("Pentest Workflow Server", "1.0.0",
			server.WithToolCapabilities(true),
			server.WithResourceCapabilities(true, true),
			server.WithRecovery(),
		),
	}

	go ds.HandleEvent()

	return ds
}

func (ds *DServer) Start(transport string, address string) {
	//start sse server
	if transport == "sse" {
		logx.Infof("Starting SSE server on :%s", address)
		sseServer := server.NewSSEServer(ds.MCPServer,
			server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
				// Add custom context values from headers
				return ctx
			}))
		if err := sseServer.Start(address); err != nil {
			log.Fatal(err)
		}
	}

	//start streamable http server
	if transport == "http" {
		logx.Infof("Starting Streamable HTTP server on :%s", address)
		httpServer := server.NewStreamableHTTPServer(ds.MCPServer)
		if err := httpServer.Start(address); err != nil {
			log.Fatal(err)
		}
	}
}

func (ds *DServer) HandleEvent() {
	for event := range ds.EventChan {
		logx.Debugf("handle event: %v", event)
		err := ds.MCPServer.SendNotificationToClient(context.Background(), "event", event)
		if err != nil {
			logx.Errorf("Failed to send notification to client: %v", err)
		}
	}
}

func GetMeta(req mcp.CallToolRequest) map[string]any {
	// First try to get threadID from Meta.AdditionalFields (recommended way)
	if req.Params.Meta != nil && req.Params.Meta.AdditionalFields != nil {
		//get all key value
		metaData := make(map[string]any)
		for k, v := range req.Params.Meta.AdditionalFields {
			metaData[k] = v
		}
		return metaData
	}
	return nil
}
