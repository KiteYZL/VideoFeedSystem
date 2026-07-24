package main

import (
	"context"
	"demo/internal/config"
	"demo/internal/db"
	apphttp "demo/internal/http"
	"demo/internal/video"
	"log"
	"os"
	"strconv"
	"time"

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
	cfg, usedDefault, err := config.LoadLocalDev(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if usedDefault {
		log.Printf("config %s not found, using default local config", configPath)
	}

	sqlDB, err := db.NewDB(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.CloseDB(sqlDB)
	if err := db.AutoMigrate(sqlDB); err != nil {
		log.Fatalf("failed to auto migrate database: %v", err)
	}

	redisClient := video.NewRedisClient(cfg.Redis)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect Redis: %v", err)
	}
	defer redisClient.Close()
	repo := video.NewRepository(sqlDB)
	cache := video.NewCache(redisClient, cfg.Video.LocalCacheCapacity, time.Duration(cfg.Video.LocalCacheTTLSeconds)*time.Second, time.Duration(cfg.Video.RedisCacheTTLSeconds)*time.Second)
	cache.SubscribeInvalidations(context.Background())
	feed := video.NewFeedStore(redisClient, cfg.Video.HotWindowMinutes, time.Duration(cfg.Video.HotSnapshotTTLSeconds)*time.Second)
	videoService := video.NewService(repo, cache, feed, cfg.Video.FeedPageSize)
	publisher, err := video.NewEventPublisher(cfg.Rabbit)
	if err != nil {
		log.Fatalf("failed to connect RabbitMQ: %v", err)
	}
	defer publisher.Close()
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	video.StartOutboxPublisher(workerCtx, repo, publisher)
	if err := video.StartProjectionConsumer(workerCtx, cfg.Rabbit, repo, videoService); err != nil {
		log.Fatalf("failed to start video projection consumer: %v", err)
	}

	r := apphttp.SetRouter(sqlDB, videoService)
	log.Printf("server is running on port %d", cfg.Server.Port)
	if err := r.Run(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
		log.Fatalf("fail to run server: %v", err)
	}
}
