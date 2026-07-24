package http

import (
	"demo/internal/account"
	appjwt "demo/internal/middleware/jwt"
	"demo/internal/video"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 创建 Gin 引擎并注册 account 路由
func SetRouter(db *gorm.DB, videoService *video.Service) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("SetTrustedProxies failed: %v", err)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	accountRepo := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepo)
	accountHandler := account.NewAccountHandler(accountService)

	// 公共 account 路由
	accountGroup := r.Group("/account")
	{
		accountGroup.POST("/register", accountHandler.CreateAccount)
		accountGroup.POST("/login", accountHandler.Login)
		accountGroup.POST("/refresh", accountHandler.Refresh)
		accountGroup.POST("/findByID", accountHandler.FindByID)
		accountGroup.POST("/findByUsername", accountHandler.FindByUsername)
	}

	// 受保护的 account 路由
	protectAccountGroup := accountGroup.Group("")
	protectAccountGroup.Use(appjwt.JWTAuth(accountRepo))
	{
		protectAccountGroup.POST("logout", accountHandler.Logout)
	}

	videoHandler := video.NewHandler(videoService)
	videoGroup := r.Group("/video")
	{
		videoGroup.POST("/feed", videoHandler.Feed)
		videoGroup.POST("/detail", videoHandler.Detail)
	}
	protectedVideoGroup := videoGroup.Group("")
	protectedVideoGroup.Use(appjwt.JWTAuth(accountRepo))
	{
		protectedVideoGroup.POST("/create", videoHandler.Create)
		protectedVideoGroup.POST("/publish", videoHandler.Publish)
		protectedVideoGroup.POST("/update", videoHandler.Update)
		protectedVideoGroup.POST("/hide", videoHandler.Hide)
		protectedVideoGroup.POST("/republish", videoHandler.Republish)
		protectedVideoGroup.POST("/delete", videoHandler.Delete)
		protectedVideoGroup.POST("/view", videoHandler.View)
		protectedVideoGroup.POST("/like/create", videoHandler.Like)
		protectedVideoGroup.POST("/like/delete", videoHandler.Unlike)
		protectedVideoGroup.POST("/like/status", videoHandler.LikeStatus)
	}

	return r
}
