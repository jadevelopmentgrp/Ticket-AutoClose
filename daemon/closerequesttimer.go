package daemon

import (
	"github.com/TicketsBot/common/sentry"
)

func (d *Daemon) SweepCloseRequestTimer()  {
	if err := d.db.CloseRequest.Cleanup(); err != nil {
		sentry.Error(err)
		return
	}

	requests, err := d.db.CloseRequest.GetCloseable()
	if err != nil {
		sentry.Error(err)
		return
	}

	for _, request := range requests {
		d.CloseRequestQueue.Push(request)
	}
}
