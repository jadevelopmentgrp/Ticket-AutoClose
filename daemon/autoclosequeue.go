package daemon

import (
	"github.com/TicketsBot/common/autoclose"
	"github.com/TicketsBot/common/sentry"
	"time"
)

type AutoCloseQueue struct {
	daemon    *Daemon
	ratelimit time.Duration
	ch        chan autoclose.Ticket
}

func NewAutoCloseQueue(daemon *Daemon, ratelimit time.Duration) *AutoCloseQueue {
	return &AutoCloseQueue{
		daemon:    daemon,
		ratelimit: ratelimit,
		ch:        make(chan autoclose.Ticket),
	}
}

func (q *AutoCloseQueue) Push(el autoclose.Ticket) {
	q.ch <- el
}

func (q *AutoCloseQueue) Listen() {
	for el := range q.ch {
		if err := autoclose.PublishMessage(q.daemon.redis, []autoclose.Ticket{el}); err != nil {
			sentry.Error(err)
		}

		time.Sleep(q.ratelimit)
	}
}
