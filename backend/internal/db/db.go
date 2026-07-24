package db

import (
	"demo/internal/account"
	"demo/internal/config"
	"demo/internal/video"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 创建gorm连接
func NewDB(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username, cfg.Passwd, cfg.Host, cfg.Port, cfg.DBname,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// 自动建表并迁移账号表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&account.Account{}, &video.Video{}, &video.Like{}, &video.ViewDedup{}, &video.OutboxEvent{}, &video.InboxEvent{})
}

// 关闭数据库连接
func CloseDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}
}
