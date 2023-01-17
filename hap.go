package hap

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"
)

type EventData interface {
	GetMetadata() Metadata
}

type handler[D EventData] struct {
	ctx     context.Context
	name    string
	fn      any
	size    uint
	timeout time.Duration
	ticker  *time.Ticker

	mu           sync.RWMutex
	bufferedData []*D
}

type RemoveHandlerFn func()

type Event[D EventData] struct {
	topic    string
	sem      *semaphore.Weighted
	handlers cmap.ConcurrentMap[string, *handler[D]]
	logger   Logger
}

func NewEvent[D EventData](logger Logger) *Event[D] {
	var d D
	topic := ToSnakeCase(
		strings.TrimRight(fmt.Sprintf("%T", d), "Data"),
	)

	return &Event[D]{
		topic:    topic,
		sem:      semaphore.NewWeighted(int64(1000)),
		handlers: cmap.New[*handler[D]](),
		logger:   logger,
	}
}

func (e *Event[D]) Topic() string {
	return e.topic
}

func (e *Event[D]) NumHandlers() int {
	return e.handlers.Count()
}

// AddHandler registers a new handler with a unique id and function fn.
//
// If id is empty, a uuidV4 will be generated instead.
//
// IMPORTANT:
//   - ctx must be property hydrated with the customer metadata.
//   - fn is not nil.
//   - fn must not block forever as we have a limited amount of goroutines in our pool.
func (e *Event[D]) AddHandler(ctx context.Context, id string, fn func(*D)) RemoveHandlerFn {
	return e.addHandler(ctx, id, fn, 0, 0)
}

// AddBufferedHandler registers a new buffered handler with a unique id, function fn, size and timeout.
//
// If id is empty, a uuidV4 will be generated instead.
// If size is zero, a default value of 100 is used.
// If timeout is zero, a default value of 1 second is used.
// Events are buffered and sent when either size or timeout is reached.
//
// IMPORTANT:
//   - ctx must be property hydrated with the customer metadata.
//   - fn is not nil.
//   - fn must not block forever as we have a limited amount of goroutines in our pool.
func (e *Event[D]) AddBufferedHandler(ctx context.Context, id string, fn func([]*D), size uint, timeout time.Duration) RemoveHandlerFn {
	return e.addHandler(ctx, id, fn, size, timeout)
}

func (e *Event[D]) addHandler(ctx context.Context, id string, fn any, size uint, timeout time.Duration) RemoveHandlerFn {
	removeHandlerFn := func() {}

	if fn == nil {
		e.logger.Error("Cannot add handler: nil function")
		return removeHandlerFn
	}

	switch fn.(type) {
	case func(*D), func([]*D):
		// All good !
	default:
		e.logger.Errorf("Cannot add handler: unknown func type: %T", fn)
		return removeHandlerFn
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if id == "" {
		id = uuid.New().String()
	}

	if size == 0 {
		size = 100
	}

	h := handler[D]{
		ctx:  ctx,
		name: fmt.Sprintf("%x", sha256.Sum256([]byte(id+"_"+GetFunctionName(fn)))),
		fn:   fn,
		size: size,
	}

	// Send buffered events every timeout.
	if fn, ok := h.fn.(func([]*D)); ok {
		if timeout == 0 {
			timeout = time.Second
		}

		h.timeout = timeout

		h.ticker = time.NewTicker(h.timeout)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-h.ticker.C:
					if err := e.sem.Acquire(ctx, 1); err != nil {
						e.logger.Debugf("failed to acquire semaphore: %v", err)
						break
					}

					go func() {
						defer e.sem.Release(1)

						h.mu.RLock()
						numBufferedEntities := len(h.bufferedData)
						h.mu.RUnlock()
						if numBufferedEntities > 0 {
							h.mu.Lock()
							bufferedEntities := DeepCopy(&h.bufferedData)
							h.bufferedData = nil
							h.mu.Unlock()
							fn(bufferedEntities)
						}
					}()
				}
			}
		}()
	}

	e.handlers.Set(h.name, &h)

	removeHandlerFn = func() {
		e.removeHandler(&h)
	}

	return removeHandlerFn
}

func (e *Event[D]) removeHandler(h *handler[D]) {
	if h.ticker != nil {
		h.ticker.Stop()
	}

	e.handlers.Remove(h.name)
}

// Trigger executes the registered handlers.
func (e *Event[D]) Trigger(ctx context.Context, data D) {
	if e.handlers.IsEmpty() {
		return
	}

	for _, h := range e.handlers.Items() {
		// Remove the handler if its context has been canceled or expired.
		if h.ctx.Err() != nil {
			e.removeHandler(h)
			continue
		}

		if err := e.sem.Acquire(h.ctx, 1); err != nil {
			e.logger.Debugf("failed to acquire semaphore: %v", err)
			continue
		}

		go func(h *handler[D]) {
			defer e.sem.Release(1)

			switch fn := h.fn.(type) {
			case func(*D):
				fn(&data)
			case func([]*D):
				// Temporarily stop the aggregation ticker while we are processing a new event.
				h.ticker.Stop()
				defer h.ticker.Reset(h.timeout)

				h.mu.Lock()
				h.bufferedData = append(h.bufferedData, &data)
				numBufferedEntities := len(h.bufferedData)
				h.mu.Unlock()
				if numBufferedEntities >= int(h.size) {
					h.mu.Lock()
					bufferedEntities := DeepCopy(&h.bufferedData)
					h.bufferedData = nil
					h.mu.Unlock()
					fn(bufferedEntities)
				}
			}
		}(h)
	}
}

// WaitForData adds a convenience handler for synchronous event subscription.
func (e *Event[D]) WaitForData(ctx context.Context, selector func(*D) bool) (<-chan *D, RemoveHandlerFn) {
	dataCh := make(chan *D, 1)

	removeHandler := e.AddHandler(ctx, "", func(data *D) {
		if selector(data) {
			dataCh <- data
		}
	})

	return dataCh, removeHandler
}

// WaitForBufferedData adds a convenience handler for synchronous buffered event subscription.
func (e *Event[D]) WaitForBufferedData(ctx context.Context, selector func(*D) bool, size uint, timeout time.Duration) (<-chan []*D, RemoveHandlerFn) {
	dataCh := make(chan []*D, 1)

	removeHandler := e.AddBufferedHandler(ctx, "", func(data []*D) {
		dataCh <- lo.Filter(data, func(item *D, _ int) bool {
			return selector(item)
		})
	}, size, timeout)

	return dataCh, removeHandler
}
