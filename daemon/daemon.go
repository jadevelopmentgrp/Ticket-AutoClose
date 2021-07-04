package daemon

import (
	"fmt"
	"github.com/TicketsBot/autoclosedaemon/config"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/go-redis/redis"
	"github.com/rxdn/gdl/rest/ratelimit"
	"time"
)

type Daemon struct {
	conf          config.Config
	db            *database.Database
	redis         *redis.Client
	premiumClient *premium.PremiumLookupClient
	Queue         *Queue
	sweepTime     time.Duration
}

func NewDaemon(conf config.Config, db *database.Database, redis *redis.Client, premiumClient *premium.PremiumLookupClient, sweepTime time.Duration) *Daemon {
	daemon := &Daemon{
		conf:          conf,
		db:            db,
		redis:         redis,
		premiumClient: premiumClient,
		sweepTime:     sweepTime,
	}

	daemon.Queue = NewQueue(daemon, time.Second*1)
	return daemon
}

func (d *Daemon) Start() {
	go d.Queue.Listen()

	for {
		d.DoSweep()
		time.Sleep(d.sweepTime)
	}
}

func (d *Daemon) DoSweep() {
	tickets, err := d.scan()
	if err != nil {
		sentry.Error(err)
		return
	}

	// make sure we don't get a huge backlog due to a worker outage
	if err := d.redis.Del("tickets:autoclose").Err(); err != nil {
		sentry.Error(err)
	}

	for _, ticket := range tickets {
		isPremium, err := d.isPremium(ticket.GuildId)
		if err != nil {
			sentry.Error(err)
			continue
		}

		if isPremium {
			d.Queue.Push(ticket)
		}
	}

	premiumCache = make(map[uint64]bool)
}

func (d *Daemon) isPremium(guildId uint64) (bool, error) {
	isPremium, ok := premiumCache[guildId]
	if ok {
		return isPremium, nil
	} else { // If not cached, figure it out
		// Find token
		whitelabelBotId, isWhitelabel, err := d.db.WhitelabelGuilds.GetBotByGuild(guildId)
		if err != nil {
			return false, err
		}

		var token, keyPrefix string

		if isWhitelabel {
			res, err := d.db.Whitelabel.GetByBotId(whitelabelBotId)
			if err != nil {
				return false, err
			}

			token = res.Token
			keyPrefix = fmt.Sprintf("ratelimiter:%d", whitelabelBotId)
		} else {
			token = d.conf.BotToken
			keyPrefix = "ratelimiter:public"
		}

		// TODO: Large sharding buckets
		ratelimiter := ratelimit.NewRateLimiter(ratelimit.NewRedisStore(d.redis, keyPrefix), 1)
		isPremium = d.premiumClient.GetTierByGuildId(guildId, true, token, ratelimiter) > premium.None
		premiumCache[guildId] = isPremium
		return isPremium, nil
	}
}
