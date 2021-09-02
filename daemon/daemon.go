package daemon

import (
	"github.com/TicketsBot/autoclosedaemon/config"
	"github.com/TicketsBot/common/premium"
	. "github.com/TicketsBot/database"
	"github.com/go-redis/redis"
	"log"
	"os"
	"time"
)

type Daemon struct {
	conf              config.Config
	db                *Database
	redis             *redis.Client
	premiumClient     *premium.PremiumLookupClient
	AutoCloseQueue    *AutoCloseQueue
	CloseRequestQueue *CloseRequestQueue

	sweepTime time.Duration
	Logger    *log.Logger
}

func NewDaemon(conf config.Config, db *Database, redis *redis.Client, premiumClient *premium.PremiumLookupClient, sweepTime time.Duration) *Daemon {
	daemon := &Daemon{
		conf:          conf,
		db:            db,
		redis:         redis,
		premiumClient: premiumClient,
		sweepTime:     sweepTime,
		Logger:        log.New(os.Stdout, "[daemon] ", 0),
	}

	daemon.AutoCloseQueue = NewAutoCloseQueue(daemon, time.Second*1)
	daemon.CloseRequestQueue = NewCloseRequestQueue(daemon, time.Second*1)

	return daemon
}

func (d *Daemon) Start() {
	go d.AutoCloseQueue.Listen()
	go d.CloseRequestQueue.Listen()

	for {
		d.SweepAutoClose()
		d.SweepCloseRequestTimer()
		time.Sleep(d.sweepTime)
	}
}