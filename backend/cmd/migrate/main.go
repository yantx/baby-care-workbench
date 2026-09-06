// migrate 数据库迁移工具：
//  1. 连接 PostgreSQL 维护库(postgres)，目标库不存在时自动创建
//  2. 按文件名顺序执行 migrations/ 目录下的 .sql 脚本
//
// 用法：go run ./cmd/migrate [-config config/config.yaml]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fatal("加载配置失败: %v", err)
	}

	// 1. 连接维护库，确保目标库存在
	adminDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	adminConn, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		fatal("连接 PostgreSQL 失败(%s:%d): %v", cfg.Host, cfg.Port, err)
	}
	var exists bool
	if err := adminConn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, cfg.DBName,
	).Scan(&exists); err != nil {
		fatal("查询数据库存在性失败: %v", err)
	}
	if !exists {
		// CREATE DATABASE 不支持参数绑定，库名来自配置文件，先校验合法性防注入
		if !isValidIdent(cfg.DBName) {
			fatal("非法数据库名: %q", cfg.DBName)
		}
		if _, err := adminConn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q ENCODING 'UTF8'`, cfg.DBName)); err != nil {
			fatal("创建数据库 %s 失败: %v", cfg.DBName, err)
		}
		fmt.Printf("✅ 已创建数据库 %s\n", cfg.DBName)
	} else {
		fmt.Printf("ℹ️  数据库 %s 已存在\n", cfg.DBName)
	}
	if err := adminConn.Close(ctx); err != nil {
		fatal("关闭维护连接失败: %v", err)
	}

	// 2. 连接目标库，按序执行迁移脚本
	targetDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode, cfg.Timezone)
	conn, err := pgx.Connect(ctx, targetDSN)
	if err != nil {
		fatal("连接目标库失败: %v", err)
	}
	defer conn.Close(context.Background())

	files, err := filepath.Glob("migrations/*.sql")
	if err != nil || len(files) == 0 {
		fatal("未找到迁移脚本(migrations/*.sql)")
	}
	sort.Strings(files)

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			fatal("读取 %s 失败: %v", f, err)
		}
		if _, err := conn.Exec(ctx, string(sqlBytes)); err != nil {
			fatal("执行 %s 失败: %v", f, err)
		}
		fmt.Printf("✅ 已执行 %s\n", f)
	}
	fmt.Println("🎉 迁移全部完成")
}

func isValidIdent(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	for _, r := range name {
		if !(r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
	os.Exit(1)
}

// loadConfig 只解析 database 段（独立轻量解析，不触发 internal/config 全局状态）
func loadConfig(path string) (*dbOnly, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	type yamlRoot struct {
		Database *dbOnly `yaml:"database"`
	}
	var root yamlRoot
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root.Database == nil {
		return nil, fmt.Errorf("配置文件缺少 database 段")
	}
	d := root.Database
	if d.Timezone == "" {
		d.Timezone = "Asia/Shanghai"
	}
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}
	return d, nil
}

type dbOnly struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	Timezone string `yaml:"timezone"`
}
