package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yantx/baby-care-workbench/backend/internal/config"
	"github.com/yantx/baby-care-workbench/backend/internal/controller"
	"github.com/yantx/baby-care-workbench/backend/internal/middleware"
	"github.com/yantx/baby-care-workbench/backend/internal/ws"
)

// Setup 初始化路由
func Setup(ctl *controller.Controller, hub *ws.Hub) *gin.Engine {
	gin.SetMode(config.Get().Server.Mode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// WebSocket（token 鉴权后升级连接）
	r.GET("/ws", ws.HandleWS(hub))

	api := r.Group("/api/v1")
	{
		// 登录（无需鉴权）
		api.POST("/auth/login", ctl.Login)

		// 业务接口（JWT 鉴权）
		auth := api.Group("", middleware.JWTAuth())
		{
			auth.GET("/user/me", ctl.Me)
			auth.PUT("/user/me", ctl.UpdateProfile)

			auth.POST("/families", ctl.CreateFamily)
			auth.POST("/families/join", ctl.JoinFamily)
			auth.GET("/families/current", ctl.CurrentFamily)

			auth.POST("/babies", ctl.CreateBaby)

			auth.POST("/records", ctl.CreateRecord)
			auth.PUT("/records/:id", ctl.UpdateRecord)
			auth.DELETE("/records/:id", ctl.DeleteRecord)
			auth.GET("/records", ctl.ListRecords)

			auth.GET("/stats/today", ctl.TodayStats)
		}
	}
	return r
}
