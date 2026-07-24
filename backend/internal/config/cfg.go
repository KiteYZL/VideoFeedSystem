package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	DB     DBConfig     `yaml:"db"`
	Redis  RedisConfig  `yaml:"redis"`
	Rabbit RabbitConfig `yaml:"rabbitmq"`
	Video  VideoConfig  `yaml:"video"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBname   string `yaml:"dbname"`
	Passwd   string `yaml:"passwd"`
	Username string `yaml:"username"`
}

type RedisConfig struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Passwd string `yaml:"passwd"`
	DB     int    `yaml:"db"`
}

type RabbitConfig struct {
	URL         string `yaml:"url"`
	Exchange    string `yaml:"exchange"`
	Queue       string `yaml:"queue"`
	ConsumerTag string `yaml:"consumer_tag"`
	RetryLimit  int    `yaml:"retry_limit"`
}

type VideoConfig struct {
	LocalCacheCapacity    int `yaml:"local_cache_capacity"`
	LocalCacheTTLSeconds  int `yaml:"local_cache_ttl_seconds"`
	RedisCacheTTLSeconds  int `yaml:"redis_cache_ttl_seconds"`
	HotWindowMinutes      int `yaml:"hot_window_minutes"`
	HotSnapshotTTLSeconds int `yaml:"hot_snapshot_ttl_seconds"`
	FeedPageSize          int `yaml:"feed_page_size"`
}

// 从 yaml 文件加载配置，最后应用环境变量覆盖
func Load(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", filename, err)
	}

	EnvOverrides(&cfg)
	return cfg, nil
}

// 用环境变量覆盖配置
func EnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}

	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}

	if v := os.Getenv("MYSQL_HOST"); v != "" {
		cfg.DB.Host = v
	}

	if v := os.Getenv("MYSQL_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.DB.Port = port
		}
	}

	if v := os.Getenv("MYSQL_USER"); v != "" {
		cfg.DB.Username = v
	}

	if v := os.Getenv("MYSQL_PASSWD"); v != "" {
		cfg.DB.Passwd = v
	}

	if v := os.Getenv("MYSQL_DB"); v != "" {
		cfg.DB.DBname = v
	}

	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = port
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Passwd = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		cfg.Rabbit.URL = v
	}
}

// 找不到配置文件时返回默认配置
func LoadLocalDev(filename string) (Config, bool, error) {
	cfg, err := Load(filename)
	if err == nil {
		return cfg, false, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return DefaultLocalConfig(), true, nil
	}
	return Config{}, false, err
}

func DefaultLocalConfig() Config {
	cfg := Config{
		Server: ServerConfig{Port: 8080},
		DB: DBConfig{
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Passwd:   "mysql123456",
			DBname:   "feedsystem",
		},
		Redis: RedisConfig{Host: "localhost", Port: 6379},
		Rabbit: RabbitConfig{
			URL: "amqp://guest:guest@localhost:5672/", Exchange: "video.events", Queue: "video.projection", ConsumerTag: "video-projection", RetryLimit: 5,
		},
		Video: VideoConfig{LocalCacheCapacity: 1024, LocalCacheTTLSeconds: 60, RedisCacheTTLSeconds: 300, HotWindowMinutes: 60, HotSnapshotTTLSeconds: 60, FeedPageSize: 10},
	}
	return cfg
}
