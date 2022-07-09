package daemon

import (
	"github.com/TicketsBot/common/closerequest"
	"github.com/TicketsBot/database"
	"time"
)

func NewCloseRequestQueue(daemon *Daemon, ratelimit time.Duration) *Queue[database.CloseRequest] {
	return NewQueue(ratelimit, func(el database.CloseRequest) error {
		return closerequest.PublishMessage(daemon.redis, el)
	})
}
