package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GoCache/cache"
	"GoCache/server"
)

func main() {
	// 创建缓存实例
	c := cache.New()

	// 设置缓存项(永不过期)
	c.Set("name", "GoCache", 0)
	c.Set("version", "0.7.0", 0)

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

	// 启动 HTTP 服务器
	fmt.Println("\nStarting HTTP server on :8080...")
	fmt.Println("API Endpoints:")
	fmt.Println("  GET    /cache/{key}    - Get cache value")
	fmt.Println("  POST   /cache/{key}    - Set cache value")
	fmt.Println("  DELETE /cache/{key}    - Delete cache")
	fmt.Println("  GET    /cache/keys     - Get all keys")
	fmt.Println("  GET    /cache/stats    - Get cache statistics")
	fmt.Println("  POST   /cache/clear    - Clear all cache")
	fmt.Println("\nPress Ctrl+C to stop the server.")

	// 创建并启动 HTTP 服务器
	hs := server.NewHTTPServerWithCache(server.HTTPServerConfig{
		Port: 8080,
	}, c)

	// 异步启动服务器
	errCh := hs.StartAsync()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down server...")
	if err := hs.Stop(); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	// 检查服务器错误
	select {
	case err := <-errCh:
		if err != nil {
			log.Fatal("Server error: ", err)
		}
	default:
	}

	fmt.Println("Server exiting")
}
