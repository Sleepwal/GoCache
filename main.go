package main

import (
	"flag"
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
	configPath := flag.String("config", "", "path to configuration file (JSON)")
	showVersion := flag.Bool("version", false, "show version info")
	flag.Parse()

	if *showVersion {
		fmt.Println("GoCache v2.1.0")
		return
	}

	var cfg *cache.Config
	if *configPath != "" {
		loaded, err := cache.LoadConfigFromFile(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}
		cfg = loaded
	} else {
		cfg = cache.DefaultConfig()
	}

	logLevel := logger.INFO
	switch cfg.LogLevel {
	case "DEBUG":
		logLevel = logger.DEBUG
	case "WARN":
		logLevel = logger.WARN
	case "ERROR":
		logLevel = logger.ERROR
	}

	logger.Init(
		logger.WithLevel(logLevel),
		logger.WithModule("main"),
	)
	defer logger.Close()

	logger.Info("GoCache starting...")

	c := cache.New(
		cache.WithMaxMemory(int(cfg.GetMaxMemory())),
	)

	c.Set("name", "GoCache", 0)
	c.Set("version", "2.0.0", 0)
	logger.Info("cache initialized", "default_keys", 2)

	c.Set("expiring_key", "value", 3*time.Second)
	evictionInterval := cfg.EvictionInterval
	if evictionInterval == 0 {
		evictionInterval = 10 * time.Second
	}
	stop := c.StartEviction(evictionInterval)
	defer stop()

	metrics := cache.NewMetricsCollector()

	httpPort := cfg.GetHTTPPort()
	respPort := cfg.GetRESPPort()

	hs := server.NewHTTPServerWithCache(server.HTTPServerConfig{
		Port: httpPort,
	}, c)
	hs.SetMetrics(metrics)
	hs.SetConfig(cfg)

	ts := server.NewTCPServerWithCache(server.TCPServerConfig{
		Port: respPort,
	}, c)
	ts.SetMetrics(metrics)
	ts.SetConfig(cfg)

	var aofLogger *cache.AOFLogger
	_ = aofLogger
	if cfg.AOFEnabled {
		aofPath := cfg.AOFPath
		if aofPath == "" {
			aofPath = "appendonly.aof"
		}

		var err error
		aofLogger, err = cache.NewAOFLoggerWithConfig(
			aofPath,
			cfg.GetAOFFsync(),
			cfg.GetAOFRewriteThreshold(),
			c,
		)
		if err != nil {
			logger.ErrorErr("failed to initialize AOF logger", err)
		} else {
			if err := aofLogger.Replay(c); err != nil {
				logger.Warn("AOF replay failed", "error", err)
			}
			stopAutoRewrite := aofLogger.StartAutoRewrite(30 * time.Second)
			defer stopAutoRewrite()
			defer aofLogger.Close()
			logger.Info("AOF persistence enabled", "path", aofPath)
		}
	}

	var snapshotStop func()
	if cfg.SnapshotEnabled {
		snapshotPath := cfg.SnapshotPath
		if snapshotPath == "" {
			snapshotPath = "dump.json"
		}

		snapshotInterval := cfg.SnapshotInterval
		if snapshotInterval == 0 {
			snapshotInterval = 5 * time.Minute
		}

		if err := c.LoadFromFile(snapshotPath); err != nil {
			logger.Warn("snapshot load skipped", "error", err)
		}

		stopCh := make(chan struct{})
		snapshotStop = func() { close(stopCh) }
		go func() {
			ticker := time.NewTicker(snapshotInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := c.SaveToFile(snapshotPath); err != nil {
						logger.ErrorErr("snapshot save failed", err)
					} else {
						logger.Info("snapshot saved", "path", snapshotPath)
					}
				case <-stopCh:
					if err := c.SaveToFile(snapshotPath); err != nil {
						logger.ErrorErr("final snapshot save failed", err)
					}
					return
				}
			}
		}()
		defer snapshotStop()
		logger.Info("snapshot persistence enabled", "path", snapshotPath, "interval", snapshotInterval.String())
	}

	tlsInfo := ""
	if cfg.IsTLSEnabled() {
		tlsInfo = " (TLS enabled)"
	}

	authInfo := ""
	if cfg.RequireAuth() {
		authInfo = " (auth required)"
	}

	fmt.Println("\nGoCache v2.0.0 - In-Memory Cache Server")
	fmt.Println("========================================")
	fmt.Printf("\nHTTP API Endpoints (port %d)%s:\n", httpPort, tlsInfo)
	fmt.Println("  GET    /cache/{key}         - Get cache value")
	fmt.Println("  POST   /cache/{key}         - Set cache value")
	fmt.Println("  DELETE /cache/{key}         - Delete cache")
	fmt.Println("  GET    /cache/keys          - Get all keys")
	fmt.Println("  GET    /cache/stats         - Get cache statistics")
	fmt.Println("  POST   /cache/clear         - Clear all cache")
	fmt.Println("  GET    /health              - Health check")
	fmt.Println("  GET    /metrics             - Prometheus metrics")
	fmt.Printf("\nRESP Protocol (port %d)%s%s:\n", respPort, tlsInfo, authInfo)
	fmt.Println("  Compatible with redis-cli and Redis clients")
	fmt.Println("  Connect: redis-cli -p 6379")
	fmt.Println("\nCLI Tool:")
	fmt.Println("  gocache-cli -h 127.0.0.1 -p 6379")

	if cfg.AOFEnabled {
		fmt.Println("\nPersistence:")
		fmt.Println("  AOF: enabled")
		fmt.Printf("  Snapshot: %s\n", map[bool]string{true: "enabled", false: "disabled"}[cfg.SnapshotEnabled])
	}

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
