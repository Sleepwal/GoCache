package main

import (
	"fmt"
	"time"

	"GoCache/cache"
)

func main() {
	// 创建缓存实例
	c := cache.New()

	// 设置缓存项(永不过期)
	c.Set("name", "GoCache", 0)
	c.Set("version", "1.0.0", 0)

	// 设置带过期时间的缓存项
	c.Set("temp", "temporary_value", 5*time.Second)

	// 获取缓存项
	if value, found := c.Get("name"); found {
		fmt.Printf("name: %v\n", value)
	}

	// 检查键是否存在
	if c.Exists("version") {
		fmt.Println("version key exists")
	}

	// 获取所有键
	keys := c.Keys()
	fmt.Printf("All keys: %v\n", keys)

	// 缓存项数量
	fmt.Printf("Count: %d\n", c.Count())

	// 删除缓存项
	c.Delete("temp")
	fmt.Println("Deleted temp key")

	// 启动定期清理(每10秒清理一次)
	c.Set("expiring_key", "value", 3*time.Second)
	stop := c.StartEviction(10 * time.Second)
	defer stop() // 程序结束时停止清理

	fmt.Println("\nGoCache demo completed successfully!")
}
