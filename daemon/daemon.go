package daemon

import (
	"github.com/TicketsBot/common/autoclose"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/go-redis/redis"
	"time"
)

type Daemon struct {
	db        *database.Database
	redis     *redis.Client
	sweepTime time.Duration
}

func NewDaemon(db *database.Database, redis *redis.Client, sweepTime time.Duration) *Daemon {
	return &Daemon{
		db:        db,
		redis:     redis,
		sweepTime: sweepTime,
	}
}

func (d *Daemon) Start() {
	for {
		d.doSweep()
		time.Sleep(d.sweepTime)
	}
}

func (d *Daemon) doSweep() {
	tickets, err := d.scan()
	if err != nil {
		sentry.Error(err)
		return
	}

	if err := autoclose.PublishMessage(d.redis, tickets); err != nil {
		sentry.Error(err)
	}
}
