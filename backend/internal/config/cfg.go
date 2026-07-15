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
	// Redis  RedisConfig
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
		if port, err := strconv.Atoi(v); err != nil {
			cfg.Server.Port = port
		}
	}

	if v := os.Getenv("MYSQL_HOST"); v != "" {
		cfg.DB.Host = v
	}

	if v := os.Getenv("MYSQL_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err != nil {
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
}

// 找不到配置文件时返回默认配置
func LoadLocalDev(filename string) (Config, bool, error) {
	cfg, err := Load(filename)
	if err != nil {
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
			Passwd:   "123456",
			DBname:   "feedsystem",
		},
	}
	return cfg
}
