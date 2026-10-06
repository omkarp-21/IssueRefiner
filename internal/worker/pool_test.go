package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeProcessor struct {
	mu        sync.Mutex
	processed []uuid.UUID
	release   chan struct{}
}

func newFakeProcessor() *fakeProcessor {
	return &fakeProcessor{release: make(chan struct{})}
}

func (fake *fakeProcessor) ProcessJob(ctx context.Context, id uuid.UUID) error {
	fake.mu.Lock()
	fake.processed = append(fake.processed, id)
	fake.mu.Unlock()
	if fake.release != nil {
		select {
		case <-fake.release:
		case <-time.After(30 * time.Second):
			return context.DeadlineExceeded
		}
	}
	return nil
}

func (fake *fakeProcessor) count() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return len(fake.processed)
}

type cancelProcessor struct {
	mu        sync.Mutex
	started   int
	cancelled int
}

func (cancel *cancelProcessor) ProcessJob(ctx context.Context, id uuid.UUID) error {
	cancel.mu.Lock()
	cancel.started++
	cancel.mu.Unlock()
	<-ctx.Done()
	cancel.mu.Lock()
	cancel.cancelled++
	cancel.mu.Unlock()
	return ctx.Err()
}

type concurrencyProcessor struct {
	mu      sync.Mutex
	current int
	max     int
	done    int
}

func (concurrency *concurrencyProcessor) ProcessJob(ctx context.Context, id uuid.UUID) error {
	concurrency.mu.Lock()
	concurrency.current++
	if concurrency.current > concurrency.max {
		concurrency.max = concurrency.current
	}
	concurrency.mu.Unlock()
	time.Sleep(50 * time.Millisecond)
	concurrency.mu.Lock()
	concurrency.current--
	concurrency.done++
	concurrency.mu.Unlock()
	return nil
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for condition")
}

func TestPoolProcessesJobs(t *testing.T) {
	proc := newFakeProcessor()
	close(proc.release)
	pool := NewPool(proc, 2, 16, 5*time.Second)
	pool.Start()
	defer pool.Stop()
	for i := 0; i < 5; i++ {
		if err := pool.Enqueue(uuid.New()); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
	}
	waitFor(t, 5*time.Second, func() bool { return proc.count() == 5 })
}

func TestPoolProcessesEachJobOnce(t *testing.T) {
	proc := newFakeProcessor()
	close(proc.release)
	pool := NewPool(proc, 3, 32, 5*time.Second)
	pool.Start()
	defer pool.Stop()
	ids := make([]uuid.UUID, 20)
	for i := range ids {
		ids[i] = uuid.New()
		if err := pool.Enqueue(ids[i]); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
	}
	waitFor(t, 5*time.Second, func() bool { return proc.count() == 20 })
	seen := make(map[uuid.UUID]int)
	proc.mu.Lock()
	for _, id := range proc.processed {
		seen[id]++
	}
	proc.mu.Unlock()
	for _, id := range ids {
		if seen[id] != 1 {
			t.Fatalf("job %s processed %d times, want 1", id, seen[id])
		}
	}
}

func TestPoolQueueFull(t *testing.T) {
	proc := newFakeProcessor()
	pool := NewPool(proc, 1, 1, 5*time.Second)
	if err := pool.Enqueue(uuid.New()); err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}
	if err := pool.Enqueue(uuid.New()); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}

func TestPoolEnqueueAfterStop(t *testing.T) {
	proc := newFakeProcessor()
	close(proc.release)
	pool := NewPool(proc, 1, 8, 5*time.Second)
	pool.Start()
	pool.Stop()
	if err := pool.Enqueue(uuid.New()); !errors.Is(err, ErrPoolStopped) {
		t.Fatalf("expected ErrPoolStopped, got %v", err)
	}
}

func TestPoolStopWaitsForActiveWork(t *testing.T) {
	proc := newFakeProcessor()
	pool := NewPool(proc, 1, 8, 5*time.Second)
	pool.Start()
	if err := pool.Enqueue(uuid.New()); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	waitFor(t, 5*time.Second, func() bool { return proc.count() == 1 })
	stopped := make(chan struct{})
	go func() {
		pool.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatalf("Stop returned before active work finished")
	case <-time.After(100 * time.Millisecond):
	}
	close(proc.release)
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatalf("Stop did not return after work finished")
	}
	if proc.count() != 1 {
		t.Fatalf("expected 1 processed job, got %d", proc.count())
	}
}

func TestPoolShutdownCancelsActiveJob(t *testing.T) {
	proc := &cancelProcessor{}
	pool := NewPool(proc, 1, 8, 5*time.Second)
	pool.Start()
	if err := pool.Enqueue(uuid.New()); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	waitFor(t, 5*time.Second, func() bool {
		proc.mu.Lock()
		defer proc.mu.Unlock()
		return proc.started == 1
	})
	if ok := pool.StopWithTimeout(5 * time.Second); !ok {
		t.Fatalf("expected Stop to return in time")
	}
	proc.mu.Lock()
	defer proc.mu.Unlock()
	if proc.cancelled != 1 {
		t.Fatalf("expected processor to observe cancellation, got %d", proc.cancelled)
	}
}

func TestPoolStopBoundedOnStuckProcessor(t *testing.T) {
	proc := newFakeProcessor()
	pool := NewPool(proc, 1, 8, 5*time.Second)
	pool.Start()
	if err := pool.Enqueue(uuid.New()); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	waitFor(t, 5*time.Second, func() bool { return proc.count() == 1 })
	if ok := pool.StopWithTimeout(200 * time.Millisecond); ok {
		t.Fatalf("expected Stop to time out on stuck work")
	}
	close(proc.release)
}

func TestPoolStartTwice(t *testing.T) {
	proc := &concurrencyProcessor{}
	pool := NewPool(proc, 2, 16, 5*time.Second)
	pool.Start()
	pool.Start()
	defer pool.Stop()
	for i := 0; i < 4; i++ {
		if err := pool.Enqueue(uuid.New()); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
	}
	waitFor(t, 5*time.Second, func() bool {
		proc.mu.Lock()
		defer proc.mu.Unlock()
		return proc.done == 4
	})
	proc.mu.Lock()
	defer proc.mu.Unlock()
	if proc.max > 2 {
		t.Fatalf("expected at most 2 concurrent workers, got %d", proc.max)
	}
}

func TestPoolEnqueueDuringShutdown(t *testing.T) {
	proc := newFakeProcessor()
	close(proc.release)
	pool := NewPool(proc, 2, 4, 5*time.Second)
	pool.Start()
	stopDone := make(chan struct{})
	go func() {
		pool.StopWithTimeout(5 * time.Second)
		close(stopDone)
	}()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				err := pool.Enqueue(uuid.New())
				if err != nil && !errors.Is(err, ErrQueueFull) && !errors.Is(err, ErrPoolStopped) {
					t.Errorf("unexpected enqueue error: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	<-stopDone
	if err := pool.Enqueue(uuid.New()); !errors.Is(err, ErrPoolStopped) {
		t.Fatalf("expected ErrPoolStopped after shutdown, got %v", err)
	}
}

func TestPoolQueuedJobsLeftOnStop(t *testing.T) {
	proc := newFakeProcessor()
	pool := NewPool(proc, 1, 8, 5*time.Second)
	if err := pool.Enqueue(uuid.New()); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if err := pool.Enqueue(uuid.New()); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if ok := pool.StopWithTimeout(time.Second); !ok {
		t.Fatalf("expected Stop to return without workers")
	}
	if proc.count() != 0 {
		t.Fatalf("expected no jobs processed, got %d", proc.count())
	}
	if err := pool.Enqueue(uuid.New()); !errors.Is(err, ErrPoolStopped) {
		t.Fatalf("expected ErrPoolStopped, got %v", err)
	}
}
