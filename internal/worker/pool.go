package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrQueueFull   = errors.New("worker queue is full")
	ErrPoolStopped = errors.New("worker pool is stopped")
)

type Processor interface {
	ProcessJob(ctx context.Context, id uuid.UUID) error
}

type Pool struct {
	processor   Processor
	workerCount int
	jobTimeout  time.Duration
	queue       chan uuid.UUID
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.Mutex
	started     bool
	stopped     bool
}

func NewPool(processor Processor, workerCount int, queueSize int, jobTimeout time.Duration) Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return Pool{
		processor:   processor,
		workerCount: workerCount,
		jobTimeout:  jobTimeout,
		queue:       make(chan uuid.UUID, queueSize),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (pool *Pool) Start() {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if pool.started || pool.stopped {
		return
	}
	pool.started = true
	for i := 0; i < pool.workerCount; i++ {
		pool.wg.Add(1)
		go pool.run(i + 1)
	}
}

func (pool *Pool) run(id int) {
	defer pool.wg.Done()
	log.Printf("worker %d started", id)
	defer log.Printf("worker %d stopped", id)
	for {
		select {
		case <-pool.ctx.Done():
			log.Printf("worker %d stopping", id)
			return
		case jobID := <-pool.queue:
			log.Printf("worker %d received job %s", id, jobID)
			pool.process(jobID)
		}
	}
}

func (pool *Pool) process(jobID uuid.UUID) {
	jobCtx, cancel := context.WithTimeout(pool.ctx, pool.jobTimeout)
	defer cancel()
	if err := pool.processor.ProcessJob(jobCtx, jobID); err != nil {
		log.Printf("worker job %s failed: %v", jobID, err)
		return
	}
	log.Printf("worker job %s completed", jobID)
}

func (pool *Pool) Enqueue(id uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if pool.stopped {
		return fmt.Errorf("%w", ErrPoolStopped)
	}
	select {
	case <-pool.ctx.Done():
		return fmt.Errorf("%w", ErrPoolStopped)
	default:
	}
	select {
	case pool.queue <- id:
		return nil
	default:
		return fmt.Errorf("%w", ErrQueueFull)
	}
}

func (pool *Pool) stop() {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if pool.stopped {
		return
	}
	pool.stopped = true
	pool.cancel()
}

func (pool *Pool) Stop() {
	pool.stop()
	pool.wg.Wait()
}

func (pool *Pool) StopWithTimeout(timeout time.Duration) bool {
	pool.stop()
	done := make(chan struct{})
	go func() {
		pool.wg.Wait()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}
