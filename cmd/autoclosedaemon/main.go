package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jadevelopmentgrp/Tickets-AutoClose/config"
	"github.com/jadevelopmentgrp/Tickets-AutoClose/daemon"
	database "github.com/jadevelopmentgrp/Tickets-Database"
	"github.com/jadevelopmentgrp/Tickets-Utilities/observability"
	"github.com/rxdn/gdl/cache"
	"go.uber.org/zap"
)

func main() {
	conf := config.ParseConfig()

	var logger *zap.Logger
	var err error
	if conf.ProductionMode {
		logger, err = zap.NewProduction(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
			zap.WrapCore(observability.ZapAdapter()),
		)
	} else {
		logger, err = zap.NewDevelopment(zap.
			AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	}

	if err != nil {
		panic(err)
	}

	logger.Debug("Connecting to database...")
	dbClient := newDatabaseClient(conf)
	logger.Debug("Connected to database, connecting to redis...")
	redisClient := newRedisClient(conf)

	logger.Debug("Starting daemon", zap.Int("sweep_time_minutes", conf.DaemonSweepTime))
	daemon.NewDaemon(
		conf,
		logger,
		dbClient,
		redisClient,
		time.Minute*time.Duration(conf.DaemonSweepTime),
	).Start()
}

func newDatabaseClient(conf config.Config) *database.Database {
	connString := fmt.Sprintf("%s?pool_max_conns=%d", conf.DatabaseUri, conf.DatabaseThreads)

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		panic(err)
	}

	return database.NewDatabase(pool)
}

func newCacheClient(conf config.Config) *cache.PgCache {
	connString := fmt.Sprintf("%s?pool_max_conns=%d&statement_cache_mode=describe", conf.CacheUri, conf.CacheThreads)

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	opts := cache.CacheOptions{
		Guilds:   true,
		Users:    true,
		Members:  true,
		Channels: true,
		Roles:    true,
	}

	client := cache.NewPgCache(pool, opts)
	return &client
}

func newRedisClient(conf config.Config) (client *redis.Client) {
	options := &redis.Options{
		Network:      "tcp",
		Addr:         conf.RedisAddress,
		Password:     conf.RedisPassword,
		PoolSize:     conf.RedisThreads,
		MinIdleConns: conf.RedisThreads,
	}

	client = redis.NewClient(options)
	if err := client.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}

	return
}
