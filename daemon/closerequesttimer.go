package daemon

import (
	"github.com/TicketsBot/common/sentry"
)

func (d *Daemon) SweepCloseRequestTimer()  {
	d.Logger.Println("starting close request sweep")

	if err := d.db.CloseRequest.Cleanup(); err != nil {
		sentry.Error(err)
		return
	}

	requests, err := d.db.CloseRequest.GetCloseable()
	if err != nil {
		sentry.Error(err)
		return
	}

	d.Logger.Printf("closing %d tickets\n", len(requests))

	for _, request := range requests {
		d.CloseRequestQueue.Push(request)
	}
}
