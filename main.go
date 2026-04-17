package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GoCache/cache"
	"GoCache/logger"
	"GoCache/server"
)

func main() {
	logger.Init(
		logger.WithLevel(logger.INFO),
		logger.WithModule("main"),
	)
	defer logger.Close()

	logger.Info("GoCache starting...")

	c := cache.New()

	c.Set("name", "GoCache", 0)
	c.Set("version", "0.7.0", 0)
	logger.Info("cache initialized", "default_keys", 2)

	c.Set("temp", "temporary_value", 5*time.Second)
	logger.Debug("set temporary key", "key", "temp", "ttl", "5s")

	if value, found := c.Get("name"); found {
		fmt.Printf("name: %v\n", value)
	}

	if c.Exists("version") {
		fmt.Println("version key exists")
	}

	keys := c.Keys()
	fmt.Printf("All keys: %v\n", keys)

	fmt.Printf("Count: %d\n", c.Count())

	c.Delete("temp")
	logger.Info("deleted key", "key", "temp")

	c.Set("expiring_key", "value", 3*time.Second)
	stop := c.StartEviction(10 * time.Second)
	defer stop()

	logger.Info("GoCache demo completed successfully")

	logger.Info("starting HTTP server", "port", 8080)
	fmt.Println("\nAPI Endpoints:")
	fmt.Println("  GET    /cache/{key}    - Get cache value")
	fmt.Println("  POST   /cache/{key}    - Set cache value")
	fmt.Println("  DELETE /cache/{key}    - Delete cache")
	fmt.Println("  GET    /cache/keys     - Get all keys")
	fmt.Println("  GET    /cache/stats    - Get cache statistics")
	fmt.Println("  POST   /cache/clear    - Clear all cache")
	fmt.Println("\nPress Ctrl+C to stop the server.")

	hs := server.NewHTTPServerWithCache(server.HTTPServerConfig{
		Port: 8080,
	}, c)

	errCh := hs.StartAsync()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	if err := hs.Stop(); err != nil {
		logger.ErrorErr("server forced to shutdown", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			logger.ErrorErr("server error", err)
		}
	default:
	}

	logger.Info("server exiting")
}
