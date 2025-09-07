package biz

import (
	"github.com/gin-gonic/gin"
	"github.com/xid-protocol/mcp-driver/examples/multi_task/server/biz/handler"
)

func RegisterRouter(r *gin.Engine) {

	apiGroup := r.Group("/api/")
	{

		//chat接口
		authGroup := apiGroup.Group("/chat")
		{
			authGroup.POST("/stream", handler.Chat)
		}

	}

}
