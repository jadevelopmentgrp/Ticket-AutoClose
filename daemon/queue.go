package daemon

import (
	"github.com/TicketsBot/common/sentry"
	"time"
)

type Queue[T any] struct {
	ratelimit time.Duration
	ch        chan T
	processor func(T) error
}

func NewQueue[T any](ratelimit time.Duration, processor func(T) error) *Queue[T] {
	return &Queue[T]{
		ratelimit: ratelimit,
		ch:        make(chan T),
		processor: processor,
	}
}

func (q *Queue[T]) Push(el T) {
	q.ch <- el
}

func (q *Queue[T]) Listen() {
	for el := range q.ch {
		if err := q.processor(el); err != nil {
			sentry.Error(err)
		}

		time.Sleep(q.ratelimit)
	}
}
