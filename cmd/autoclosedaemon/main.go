package main

import (
	"context"
	"fmt"
	"github.com/TicketsBot/autoclosedaemon/config"
	"github.com/TicketsBot/autoclosedaemon/daemon"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rxdn/gdl/cache"
	"time"
)

func main() {
	conf := config.ParseConfig()

	if err := sentry.Initialise(sentry.Options{
		Dsn:     conf.SentryDSN,
		Project: "autoclosedaemon",
		Debug:   conf.SentryDSN == "",
	}); err != nil {
		fmt.Println(err.Error())
	}

	dbClient := newDatabaseClient(conf)
	redisClient := newRedisClient(conf)
	cacheClient := newCacheClient(conf)
	premiumClient := newPremiumClient(conf, redisClient, cacheClient, dbClient)

	daemon := daemon.NewDaemon(conf, dbClient, redisClient, premiumClient, time.Minute*time.Duration(conf.DaemonSweepTime))
	daemon.Start()
}

func newDatabaseClient(conf config.Config) *database.Database {
	connString := fmt.Sprintf("%s?pool_max_conns=%d", conf.DatabaseUri, conf.DatabaseThreads)

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		sentry.Error(err)
		panic(err)
	}

	return database.NewDatabase(pool)
}

func newCacheClient(conf config.Config) *cache.PgCache {
	connString := fmt.Sprintf("%s?pool_max_conns=%d&statement_cache_mode=describe", conf.CacheUri, conf.CacheThreads)

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		sentry.Error(err)
		panic(err)
	}

	pool, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		sentry.Error(err)
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
		sentry.Error(err)
		panic(err)
	}

	return
}

func newPremiumClient(conf config.Config, redisClient *redis.Client, cacheClient *cache.PgCache, databaseClient *database.Database) *premium.PremiumLookupClient {
	patreonClient := premium.NewPatreonClient(conf.PatreonProxyUrl, conf.PatreonProxyKey)
	return premium.NewPremiumLookupClient(patreonClient, redisClient, cacheClient, databaseClient)
}
