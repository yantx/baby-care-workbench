package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 全局配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	JWT      JWTConfig      `yaml:"jwt"`
	Wechat   WechatConfig   `yaml:"wechat"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	Timezone string `yaml:"timezone"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode, d.Timezone,
	)
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"` // 有效期（小时）
}

type WechatConfig struct {
	AppID  string `yaml:"appid"`
	Secret string `yaml:"secret"`
}

// MockLogin 微信配置缺失时启用 Mock 登录（本地开发调试用）
func (w WechatConfig) MockLogin() bool {
	return w.AppID == "" || w.Secret == ""
}

var cfg *Config

// envOverride 环境变量覆盖（优先级高于配置文件，生产环境用它注入密码等敏感信息）
// 支持：BABY_SERVER_PORT / BABY_DB_HOST / BABY_DB_PORT / BABY_DB_USER / BABY_DB_PASSWORD /
//
//	BABY_DB_NAME / BABY_REDIS_HOST / BABY_REDIS_PORT / BABY_REDIS_PASSWORD / BABY_REDIS_DB
func envOverride(c *Config) {
	if v := os.Getenv("BABY_SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Server.Port = p
		}
	}
	if v := os.Getenv("BABY_DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("BABY_DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Database.Port = p
		}
	}
	if v := os.Getenv("BABY_DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("BABY_DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("BABY_DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := os.Getenv("BABY_REDIS_HOST"); v != "" {
		c.Redis.Host = v
	}
	if v := os.Getenv("BABY_REDIS_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Redis.Port = p
		}
	}
	if v := os.Getenv("BABY_REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}
	if v := os.Getenv("BABY_REDIS_DB"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Redis.DB = p
		}
	}
	if v := os.Getenv("BABY_JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	}
}

// Load 加载配置文件，path 为空时使用默认路径
func Load(path string) (*Config, error) {
	if path == "" {
		path = "config/config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	c := &Config{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	if c.JWT.ExpireHours <= 0 {
		c.JWT.ExpireHours = 720
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	envOverride(c)
	cfg = c
	return c, nil
}

// Get 获取已加载的全局配置
func Get() *Config {
	return cfg
}
