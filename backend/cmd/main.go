package main

import (
	"demo/internal/config"
	"demo/internal/db"
	apphttp "demo/internal/http"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env no found; continuing")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	// 加载配置
	cfg, usedDefault, err := config.LoadLocalDev(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if usedDefault {
		log.Printf("config %s not found, using default local config", configPath)
	} else {
		log.Printf("config loaded from %s", configPath)
	}

	// 初始化 MySQL
	sqlDB, err := db.NewDB(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.CloseDB(sqlDB)

	if err := db.AutoMigrate(sqlDB); err != nil {
		log.Fatalf("failed to auto migrate database: %v", err)
	}

	// 装配路由并启动服务
	r := apphttp.SetRouter(sqlDB)
	log.Println("server is running on port %d", cfg.Server.Port)
	if err := r.Run(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
		log.Fatalf("fail to run server: %v", err)
	}
}
