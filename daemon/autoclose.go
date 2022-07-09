package daemon

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/rxdn/gdl/rest/ratelimit"
	"time"
)

func (d *Daemon) SweepAutoClose() {
	d.Logger.Println("starting autoclose sweep")
	tickets, err := d.scan()
	if err != nil {
		sentry.Error(err)
		return
	}

	// make sure we don't get a huge backlog due to a worker outage
	if err := d.redis.Del(context.Background(), "tickets:autoclose").Err(); err != nil {
		sentry.Error(err)
	}

	d.Logger.Printf("closing %d tickets\n", len(tickets))

	for _, ticket := range tickets {
		isPremium, err := d.isPremium(ticket.GuildId)
		if err != nil {
			sentry.Error(err)
			continue
		}

		if isPremium {
			// Convert message ID to timestamp for debug logging
			if ticket.LastMessageId == nil {
				d.Logger.Printf("Closing %d ticket #%d (no messages)\n", ticket.GuildId, ticket.TicketId)
			} else {
				shifted := *ticket.LastMessageId >> 22
				lastMessageTime := time.UnixMilli(int64(shifted + 1420070400000))

				d.Logger.Printf("Closing %d ticket #%d (last message time: %s)\n", ticket.GuildId, ticket.TicketId, lastMessageTime.String())
			}

			d.AutoCloseQueue.Push(ticket)
		} else {
			d.Logger.Printf("Guild %d (ticket %d) does not have premium, so resetting autoclose settings", ticket.GuildId, ticket.TicketId)

			if err := d.db.AutoClose.Reset(ticket.GuildId); err != nil {
				sentry.Error(err)
				continue
			}
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
		premiumTier, err := d.premiumClient.GetTierByGuildId(guildId, true, token, ratelimiter)
		if err == nil {
			premiumCache[guildId] = premiumTier > premium.None
		}

		return premiumTier > premium.None, err
	}
}

