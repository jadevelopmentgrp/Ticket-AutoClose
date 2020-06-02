package main

import (
	"context"
	"fmt"
	"github.com/TicketsBot/autoclosedaemon/daemon"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/go-redis/redis"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
	"strconv"
	"time"
)

func main() {
	if err := sentry.Initialise(sentry.Options{
		Dsn:     os.Getenv("SENTRY_DSN"),
		Project: "autoclosedaemon",
	}); err != nil {
		fmt.Println(err.Error())
	}

	sweepTime, err := strconv.Atoi(os.Getenv("SWEEP_TIME"))
	if err != nil {
		sentry.Error(err)
		panic(err)
	}

	daemon := daemon.NewDaemon(newDatabaseClient(), newRedisClient(), time.Minute * time.Duration(sweepTime))
	daemon.Start()
}

func newDatabaseClient() *database.Database {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?pool_max_conns=%S",
		os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASSWORD"),
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_NAME"),
		os.Getenv("DATABASE_THREADS"),
	)

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		sentry.Error(err)
		panic(err)
	}

	return database.NewDatabase(pool)
}

func newRedisClient() (client *redis.Client) {
	threads, err := strconv.Atoi(os.Getenv("REDIS_THREADS"))
	if err != nil {
		panic(err)
	}

	options := &redis.Options{
		Network:      "tcp",
		Addr:         os.Getenv("REDIS_ADDR"),
		Password:     os.Getenv("REDIS_PASSWD"),
		PoolSize:     threads,
		MinIdleConns: threads,
	}

	client = redis.NewClient(options)
	if err := client.Ping().Err(); err != nil {
		sentry.Error(err)
		panic(err)
	}

	return
}
