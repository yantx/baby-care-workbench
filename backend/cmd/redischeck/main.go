package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "192.168.1.8:6379",
		Password: "redis@6379",
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Println("PING FAILED:", err)
		return
	}
	fmt.Println("PING:", pong)

	if err := rdb.Set(ctx, "baby_care:smoke", "ok", 30*time.Second).Err(); err != nil {
		fmt.Println("SET FAILED:", err)
		return
	}
	val, err := rdb.Get(ctx, "baby_care:smoke").Result()
	if err != nil {
		fmt.Println("GET FAILED:", err)
		return
	}
	fmt.Println("GET baby_care:smoke =", val)
	fmt.Println("Redis 连接验证通过")
}
