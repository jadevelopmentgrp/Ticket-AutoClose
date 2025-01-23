package daemon

import (
	"time"

	database "github.com/jadevelopmentgrp/Tickets-Database"
	"github.com/jadevelopmentgrp/Tickets-Utilities/closerequest"
	"go.uber.org/zap"
)

func NewCloseRequestQueue(daemon *Daemon, ratelimit time.Duration) *Queue[database.CloseRequest] {
	return NewQueue(daemon.logger, ratelimit, func(el database.CloseRequest) error {
		daemon.logger.Info(
			"Publishing ticket close to workers (close request)",
			zap.Uint64("guild", el.GuildId),
			zap.Int("ticket", el.TicketId),
		)
		return closerequest.PublishMessage(daemon.redis, el)
	})
}
