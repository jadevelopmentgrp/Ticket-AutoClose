package daemon

import (
	"github.com/TicketsBot/common/autoclose"
	"github.com/TicketsBot/common/sentry"
	"time"
)

type Queue struct {
	daemon    *Daemon
	ratelimit time.Duration
	ch        chan autoclose.Ticket
}

func NewQueue(daemon *Daemon, ratelimit time.Duration) *Queue {
	return &Queue{
		daemon:    daemon,
		ratelimit: ratelimit,
		ch:        make(chan autoclose.Ticket),
	}
}

func (q *Queue) Push(el autoclose.Ticket) {
	q.ch <- el
}

func (q *Queue) Listen() {
	for el := range q.ch {
		if err := autoclose.PublishMessage(q.daemon.redis, []autoclose.Ticket{el}); err != nil {
			sentry.Error(err)
		}

		time.Sleep(q.ratelimit)
	}
}
