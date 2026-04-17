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
	c.Set("version", "1.1.0", 0)
	logger.Info("cache initialized", "default_keys", 2)

	c.Set("expiring_key", "value", 3*time.Second)
	stop := c.StartEviction(10 * time.Second)
	defer stop()

	httpPort := 8080
	respPort := 6379

	hs := server.NewHTTPServerWithCache(server.HTTPServerConfig{
		Port: httpPort,
	}, c)

	ts := server.NewTCPServerWithCache(server.TCPServerConfig{
		Port: respPort,
	}, c)

	fmt.Println("\nGoCache v1.1.0 - In-Memory Cache Server")
	fmt.Println("========================================")
	fmt.Println("\nHTTP API Endpoints (port 8080):")
	fmt.Println("  GET    /cache/{key}         - Get cache value")
	fmt.Println("  POST   /cache/{key}         - Set cache value")
	fmt.Println("  DELETE /cache/{key}         - Delete cache")
	fmt.Println("  GET    /cache/keys          - Get all keys")
	fmt.Println("  GET    /cache/stats         - Get cache statistics")
	fmt.Println("  POST   /cache/clear         - Clear all cache")
	fmt.Println("  GET    /cache/health        - Health check")
	fmt.Println("\nRESP Protocol (port 6379):")
	fmt.Println("  Compatible with redis-cli and Redis clients")
	fmt.Println("  Connect: redis-cli -p 6379")
	fmt.Println("\nCLI Tool:")
	fmt.Println("  gocache-cli -h 127.0.0.1 -p 6379")
	fmt.Println("\nPress Ctrl+C to stop the server.")

	httpErrCh := hs.StartAsync()
	respErrCh := ts.StartAsync()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down servers...")
	if err := ts.Stop(); err != nil {
		logger.ErrorErr("RESP server shutdown error", err)
	}
	if err := hs.Stop(); err != nil {
		logger.ErrorErr("HTTP server shutdown error", err)
	}

	select {
	case err := <-httpErrCh:
		if err != nil {
			logger.ErrorErr("HTTP server error", err)
		}
	default:
	}

	select {
	case err := <-respErrCh:
		if err != nil {
			logger.ErrorErr("RESP server error", err)
		}
	default:
	}

	logger.Info("server exiting")
}
