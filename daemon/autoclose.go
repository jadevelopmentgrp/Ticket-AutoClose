package daemon

import (
	"fmt"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/rxdn/gdl/rest/ratelimit"
)

func (d *Daemon) SweepAutoClose() {
	d.Logger.Println("starting autoclose sweep")
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

		// TODO: Need isPremium to return error, so that we can purge settings
		if isPremium {
			d.Logger.Printf("Closing %d ticket #%d\n", ticket.GuildId, ticket.TicketId)
			d.AutoCloseQueue.Push(ticket)
		}
	}

	premiumCache = make(map[uint64]bool)
	d.Logger.Println("done")
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
		premiumTier, err := d.premiumClient.GetTierByGuildId(guildId, true, token, ratelimiter)
		if err == nil {
			premiumCache[guildId] = premiumTier > premium.None
		}

		return premiumTier > premium.None, err
	}
}

