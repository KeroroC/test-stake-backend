package main

import (
	"context"
	"fmt"
	"log"
	"test-stake-backend/internal/api"
	"test-stake-backend/internal/config"
	"test-stake-backend/internal/listener"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func main() {
	// 1.加载配置
	cfg := config.Load()

	// 2.初始化数据库
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
	// 自动迁移
	if err := db.AutoMigrate(&models.Contract{}, &models.StakedEvent{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 3.初始化Redis连接
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Fatalf("Failed to close redis: %v", err)
		}
	}()

	// 4.初始化eth连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rpcClient, err := ethclient.DialContext(ctx, cfg.ETHConfig.RPCUrl)
	if err != nil {
		log.Fatalf("Failed to conncect ETH client: %v", err)
	}
	defer rpcClient.Close()

	stakedEventRepo, err := repository.NewStakedEventRepository(db)
	if err != nil {
		log.Fatalf("Failed to create staked event repository: %v", err)
	}
	contractEventListener, err := listener.NewContractEventListener(cfg.ETHConfig.WSUrl, cfg.ETHConfig.StakeAddress)
	if err != nil {
		log.Fatalf("Failed to create contract event listener: %v", err)
	}
	stakedEventHandler, err := listener.NewStakedEventLogHandler(stakedEventRepo)
	if err != nil {
		log.Fatalf("Failed to create staked event handler: %v", err)
	}
	if err := contractEventListener.Register(stakedEventHandler); err != nil {
		log.Fatalf("Failed to register staked event handler: %v", err)
	}
	go contractEventListener.Start(context.Background())

	// 启动gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	if err := api.RegisterRoutes(r, db); err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}
	if err := r.Run(); err != nil {
		log.Fatalf("Failed to run server: %s", err)
	}
}
