package daemon

import (
	"github.com/TicketsBot/common/closerequest"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"time"
)

type CloseRequestQueue struct {
	daemon    *Daemon
	ratelimit time.Duration
	ch        chan database.CloseRequest
}

func NewCloseRequestQueue(daemon *Daemon, ratelimit time.Duration) *CloseRequestQueue {
	return &CloseRequestQueue{
		daemon:    daemon,
		ratelimit: ratelimit,
		ch:        make(chan database.CloseRequest),
	}
}

func (q *CloseRequestQueue) Push(el database.CloseRequest) {
	q.ch <- el
}

func (q *CloseRequestQueue) Listen() {
	for el := range q.ch {
		if err := closerequest.PublishMessage(q.daemon.redis, el); err != nil {
			sentry.Error(err)
		}

		time.Sleep(q.ratelimit)
	}
}
