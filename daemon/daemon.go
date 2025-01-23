package daemon

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jadevelopmentgrp/Tickets-AutoClose/config"
	database "github.com/jadevelopmentgrp/Tickets-Database"
	"github.com/jadevelopmentgrp/Tickets-Utilities/autoclose"
	"go.uber.org/zap"
)

type Daemon struct {
	conf              config.Config
	logger            *zap.Logger
	db                *database.Database
	redis             *redis.Client
	AutoCloseQueue    *Queue[autoclose.Ticket]
	CloseRequestQueue *Queue[database.CloseRequest]

	sweepTime time.Duration
}

func NewDaemon(
	conf config.Config,
	logger *zap.Logger,
	db *database.Database,
	redis *redis.Client,
	sweepTime time.Duration,
) *Daemon {
	daemon := &Daemon{
		conf:      conf,
		logger:    logger,
		db:        db,
		redis:     redis,
		sweepTime: sweepTime,
	}

	daemon.AutoCloseQueue = NewAutoCloseQueue(daemon, time.Second*1)
	daemon.CloseRequestQueue = NewCloseRequestQueue(daemon, time.Second*1)

	return daemon
}

func (d *Daemon) Start() {
	go d.AutoCloseQueue.Listen()
	go d.CloseRequestQueue.Listen()

	for {
		select {
		case <-time.After(d.sweepTime):
			d.logger.Debug("Starting run")
			d.doOne()
			d.logger.Debug("Finished run")
		}
	}
}

func (d *Daemon) doOne() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10) // TODO: Don't hardcode
	defer cancel()

	d.SweepAutoClose(ctx)
	d.SweepCloseRequestTimer(ctx)
}
