package daemon

import (
	"github.com/TicketsBot/common/autoclose"
	"time"
)

func NewAutoCloseQueue(daemon *Daemon, ratelimit time.Duration) *Queue[autoclose.Ticket] {
	return NewQueue[autoclose.Ticket](ratelimit, func(el autoclose.Ticket) error {
		return autoclose.PublishMessage(daemon.redis, []autoclose.Ticket{el})
	})
}
