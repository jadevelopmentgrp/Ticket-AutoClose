package daemon

import (
	"context"
	"time"

	"github.com/jadevelopmentgrp/Tickets-Utilities/collections"
	"go.uber.org/zap"
)

var (
	premiumCache  = make(map[uint64]bool)
	botNotInGuild = collections.NewSet[uint64]()
)

func (d *Daemon) SweepAutoClose(ctx context.Context) {
	d.logger.Debug("Starting autoclose sweep")
	tickets, err := d.scan()
	if err != nil {
		d.logger.Error("Error querying database for tickets to close (autoclose)", zap.Error(err))
		return
	}

	// make sure we don't get a huge backlog due to a worker outage
	if err := d.redis.Del(context.Background(), "tickets:autoclose").Err(); err != nil {
		d.logger.Error("Error clearing autoclose Redis queue", zap.Error(err))
		return
	}

	d.logger.Debug("Closing tickets (autoclose)", zap.Int("count", len(tickets)))

	for _, ticket := range tickets {
		if notInGuild := botNotInGuild.Contains(ticket.GuildId); notInGuild {
			if err := d.db.AutoCloseExclude.Exclude(ctx, ticket.GuildId, ticket.TicketId); err != nil {
				d.logger.Error(
					"Error excluding ticket from autoclose",
					zap.Error(err),
					zap.Uint64("guild", ticket.GuildId),
					zap.Int("ticket", ticket.TicketId),
				)
			}

			continue
		}

		// Convert message ID to timestamp for debug logging
		if ticket.LastMessageId == nil {
			d.logger.Info(
				"Queueing ticket close (no messages)",
				zap.Uint64("guild", ticket.GuildId),
				zap.Int("ticket", ticket.TicketId),
			)
		} else {
			shifted := *ticket.LastMessageId >> 22
			lastMessageTime := time.UnixMilli(int64(shifted + 1420070400000))

			d.logger.Info(
				"Queueing ticket close (timeout elapsed)",
				zap.Uint64("guild", ticket.GuildId),
				zap.Int("ticket", ticket.TicketId),
				zap.Time("last_message", lastMessageTime),
			)
		}

		d.AutoCloseQueue.Push(ticket)
	}

	premiumCache = make(map[uint64]bool)
	botNotInGuild = collections.NewSet[uint64]()
}
