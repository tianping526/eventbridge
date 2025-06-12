package informer

import (
	"container/heap"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultDeltaQueueSize = 1024
	DefaultRetryQueueSize = 1024
	DefaultWorkerNum      = 20
)

type option struct {
	deltaQueueSize int
	retryQueueSize int
	workerNum      int
}

type Option func(*option)

func WithDeltaQueueSize(size int) Option {
	return func(o *option) {
		o.deltaQueueSize = size
	}
}

func WithRetryQueueSize(size int) Option {
	return func(o *option) {
		o.retryQueueSize = size
	}
}

func WithWorkerNum(num int) Option {
	return func(o *option) {
		o.workerNum = num
	}
}

type Informer struct {
	log *log.Helper

	opt *option

	reflector Reflector
	handler   Handler

	eg         *errgroup.Group
	deltaQueue chan string
	retryQueue chan string

	closeCh chan struct{}
}

func NewInformer(logger log.Logger, reflector Reflector, handler Handler, opts ...Option) *Informer {
	o := &option{
		deltaQueueSize: DefaultDeltaQueueSize,
		retryQueueSize: DefaultRetryQueueSize,
		workerNum:      DefaultWorkerNum,
	}
	for _, opt := range opts {
		opt(o)
	}

	eg := new(errgroup.Group)
	eg.SetLimit(o.workerNum + 2)

	return &Informer{
		log: log.NewHelper(log.With(
			logger,
			"module", "informer",
			"caller", log.DefaultCaller,
		)),

		opt: o,

		reflector: reflector,
		handler:   handler,

		eg:         eg,
		deltaQueue: make(chan string, o.deltaQueueSize),
		retryQueue: make(chan string, o.retryQueueSize),

		closeCh: make(chan struct{}),
	}
}

func (i *Informer) WatchAndHandle() {
	// consume deltaQueue
	i.eg.Go(func() error {
		for key := range i.deltaQueue {
			i.eg.Go(func() error {
				err := i.handler.Handle(key)
				if err != nil {
					i.log.Errorf("handle key %s error: %v", key, err)
					i.retryQueue <- key
				}
				return nil
			})
		}
		close(i.retryQueue)
		return nil
	})

	// consume retryQueue
	i.eg.Go(func() error {
		pq := make(PriorityQueue, 0)
		items := make(map[string]*Item)
		expiredTime := time.Now().Add(24 * time.Hour)
		timer := time.NewTimer(24 * time.Hour)

		for {
			select {
			case key, ok := <-i.retryQueue:
				if !ok { // retryQueue closed
					timer.Stop()
					return nil
				}

				item, ok := items[key]
				if !ok { // new item
					bo := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(24 * time.Hour))
					next := bo.NextBackOff()
					priority := time.Now().Add(next)
					item = &Item{
						value:    key,
						bo:       bo,
						priority: priority,
					}
					items[key] = item
					heap.Push(&pq, item)
					if pq[0].priority != expiredTime {
						expiredTime = pq[0].priority
						timer.Reset(time.Until(pq[0].priority))
					}
					continue
				}

				// update item
				item.bo.Reset()
				next := item.bo.NextBackOff()
				priority := time.Now().Add(next)
				pq.update(item, key, priority)
				if pq[0].priority != expiredTime {
					expiredTime = pq[0].priority
					timer.Reset(time.Until(pq[0].priority))
				}
			case <-timer.C:
				if len(pq) == 0 { // empty queue
					expiredTime = time.Now().Add(24 * time.Hour)
					timer.Reset(24 * time.Hour)
					continue
				}

				item := heap.Pop(&pq).(*Item)
				key := item.value
				err := i.handler.Handle(key)
				if err != nil {
					next := item.bo.NextBackOff()
					if next == backoff.Stop { // stop retry
						delete(items, key)
						i.log.Errorf("handle key %s error: %v, stop retry", key, err)

						if len(pq) == 0 {
							expiredTime = time.Now().Add(24 * time.Hour)
							timer.Reset(24 * time.Hour)
							continue
						}

						expiredTime = pq[0].priority
						timer.Reset(time.Until(pq[0].priority))
						continue
					}

					// retry
					priority := time.Now().Add(next)
					item.priority = priority
					heap.Push(&pq, item)
					expiredTime = pq[0].priority
					timer.Reset(time.Until(pq[0].priority))
					i.log.Errorf("handle key %s error: %v, retry after %v", key, err, next)
					continue
				}
				delete(items, key)

				if len(pq) == 0 {
					expiredTime = time.Now().Add(24 * time.Hour)
					timer.Reset(24 * time.Hour)
					continue
				}

				expiredTime = pq[0].priority
				timer.Reset(time.Until(pq[0].priority))
			}
		}
	})

	for {
		select {
		case <-i.closeCh:
			close(i.deltaQueue)
			_ = i.eg.Wait()
			return
		default:
			// watch
			keys, err := i.reflector.Watch()
			if err != nil {
				if IsReflectorClosedError(err) {
					continue
				}
				i.log.Errorf("watch error: %v", err)
				err = backoff.Retry(func() error {
					keys, err = i.reflector.Watch()
					if err != nil {
						if IsReflectorClosedError(err) {
							return backoff.Permanent(err)
						}
						i.log.Errorf("retry watch error: %v", err)
					}
					return err
				}, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(24*time.Hour)))
				if err != nil {
					if !IsReflectorClosedError(err) {
						i.log.Errorf("retry watch failed after 24 hours: %v", err)
					}
					continue
				}
			}

			// enqueue
			for _, key := range keys {
				i.deltaQueue <- key
			}
		}
	}
}

func (i *Informer) Close() {
	close(i.closeCh)
	err := i.reflector.Close()
	if err != nil {
		i.log.Errorf("close reflector failed: %v", err)
	}
}
