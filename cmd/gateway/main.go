// Package main is the entrypoint for the High-Throughput API Gateway.
// It wires all dependencies and starts the HTTP server.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/raizel/gateway/internal/cache"
	"github.com/raizel/gateway/internal/config"
	"github.com/raizel/gateway/internal/encoding"
	"github.com/raizel/gateway/internal/middleware"
	"github.com/raizel/gateway/internal/repository"
	"github.com/raizel/gateway/internal/router"
	"github.com/raizel/gateway/internal/server"
	"github.com/raizel/gateway/internal/service"
	"github.com/raizel/gateway/internal/telemetry"
)

func main() {
	// Parse CLI flags.
	configPath := flag.String("config", "", "Path to config file (default: configs/app.dev.yaml)")
	flag.Parse()

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize structured logger.
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("starting gateway",
		zap.Int("port", cfg.Server.Port),
		zap.String("env", cfg.Telemetry.ServiceName),
	)

	// --- Connect to PostgreSQL ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.DSN())
	if err != nil {
		logger.Fatal("invalid database DSN", zap.Error(err))
	}
	poolConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.Database.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.Database.MaxLifetime

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		logger.Fatal("PostgreSQL ping failed", zap.Error(err))
	}
	logger.Info("connected to PostgreSQL")

	// --- Connect to Redis ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	logger.Info("connected to Redis")

	// --- Initialize tracing (optional) ---
	if cfg.Telemetry.TracingEnabled {
		shutdownTracer, err := telemetry.InitTracer(ctx, cfg.Telemetry.ServiceName, cfg.Telemetry.OTLPEndpoint)
		if err != nil {
			logger.Warn("failed to initialize tracing", zap.Error(err))
		} else {
			defer shutdownTracer(context.Background())
			logger.Info("tracing initialized")
		}
	}

	// --- Build dependency graph ---
	metrics := telemetry.NewMetrics()
	encoder := encoding.NewEncoder(cfg.ShortURL.CodeLength)
	bloomFilter := cache.NewBloomFilter(1_000_000, 0.01)
	urlRepo := repository.NewPostgresURLRepo(dbPool)
	urlCache := cache.NewRedisCache(redisClient)
	circuitBreaker := middleware.NewCircuitBreaker("database")

	urlService := service.NewURLService(
		urlRepo,
		urlCache,
		bloomFilter,
		encoder,
		cfg,
		metrics,
		logger,
	)

	// --- Wire router ---
	r := router.New(&router.Deps{
		URLService:     urlService,
		Repo:           urlRepo,
		Cache:          urlCache,
		RedisClient:    redisClient,
		CircuitBreaker: circuitBreaker,
		Config:         cfg,
		Metrics:        metrics,
		Logger:         logger,
	})

	// --- Start server ---
	srv := server.New(r, &cfg.Server, logger)
	if err := srv.Start(); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
