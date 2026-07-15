package http

import (
	"demo/internal/account"
	appjwt "demo/internal/middleware/jwt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 创建 Gin 引擎并注册 account 路由
func SetRouter(db *gorm.DB) *gin.Engine {
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

	return r
}
