package main

import (
	"fmt"
	"log"
	"test-stake-backend/internal/api"
	"test-stake-backend/internal/config"
	"test-stake-backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	dbConfig := cfg.Database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),

		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
			NoLowerCase:   false,
		},
	})
	if err != nil {
		log.Fatalf("failed to connect database, %v", err)
	}

	if err := db.AutoMigrate(&models.Contract{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 启动gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	api.RegisterRoutes(r)
	if err := r.Run(); err != nil {
		log.Fatalf("Failed to run server: %s", err)
	}
}
