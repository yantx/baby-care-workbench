package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/yantx/baby-care-workbench/backend/internal/config"
	"github.com/yantx/baby-care-workbench/backend/internal/controller"
	"github.com/yantx/baby-care-workbench/backend/internal/model"
	"github.com/yantx/baby-care-workbench/backend/internal/repository"
	"github.com/yantx/baby-care-workbench/backend/internal/router"
	"github.com/yantx/baby-care-workbench/backend/internal/service"
	"github.com/yantx/baby-care-workbench/backend/internal/ws"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 连接 PostgreSQL
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: %v\n请确认: 1) PG 服务已启动  2) 数据库 %s 已创建  3) config/config.yaml 密码已填写",
			err, cfg.Database.DBName)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移（Phase 1 使用 GORM AutoMigrate；生产环境改用 migrations/*.sql 受控迁移）
	if err := db.AutoMigrate(
		&model.User{}, &model.Family{}, &model.FamilyMember{},
		&model.Baby{}, &model.CareRecord{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Printf("数据库连接成功: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 3. Redis（可选，不可达仅告警）
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr(), Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[warn] Redis 不可达(%v)，Phase 1 单实例部署可正常运行，多实例部署需要 Redis", err)
	} else {
		log.Printf("Redis 连接成功: %s", cfg.Redis.Addr())
	}
	cancel()

	// 4. 初始化各层
	hub := ws.NewHub()
	go hub.Run()
	repo := repository.New(db)
	svc := service.New(repo, hub)
	ctl := controller.New(svc)

	// 5. HTTP 服务
	r := router.Setup(ctl, hub)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		log.Printf("服务启动: http://127.0.0.1:%d  (WebSocket: ws://127.0.0.1:%d/ws)", cfg.Server.Port, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 6. 优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = sqlDB.Close()
	log.Println("服务已退出")
}
