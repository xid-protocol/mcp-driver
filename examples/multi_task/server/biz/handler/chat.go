package handler

import (
	"fmt"
	"net/http"

	"github.com/colin-404/logx"
	"github.com/gin-gonic/gin"
	"github.com/xid-protocol/common"
	"github.com/xid-protocol/mcp-driver/connection/ssepool"
)

var messageChan = make(chan string)

type chatRequest struct {
	Content  string `json:"message"`
	ThreadID string `json:"thread_id"`
}

func Chat(c *gin.Context) {

	var chatReq chatRequest
	c.ShouldBindJSON(chatReq)

	logx.Info(chatReq)
	//判断threadID是否存在
	threadID := chatReq.ThreadID
	if threadID == "" {
		threadID = fmt.Sprintf("thread_%s", common.GenerateID())
	}
	conn, err := ssepool.Attach(threadID, c.Writer)
	if err != nil {
		logx.Errorf("Attach error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer conn.Cancel()

	//开始处理消息内容
	go handleMessage(chatReq.Content)

	c.JSON(http.StatusOK, gin.H{"threadID": threadID})
	<-c.Done()
}

func handleMessage(message string) {

}
