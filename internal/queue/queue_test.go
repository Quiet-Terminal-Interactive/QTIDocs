package queue

import (
	"context"
	"sync"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

func TestQueue_SingleJobRuns(t *testing.T) {
	var mu sync.Mutex
	var ran []string
	done := make(chan struct{})

	q := New(func(ctx context.Context, job string) {
		mu.Lock()
		ran = append(ran, job)
		mu.Unlock()
		close(done)
	})
	q.Enqueue(context.Background(), "k", "A")

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for job to run")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(ran) != 1 || ran[0] != "A" {
		t.Errorf("ran = %v, want [A]", ran)
	}
}

func TestQueue_DifferentKeysRunIndependently(t *testing.T) {
	var mu sync.Mutex
	ran := map[string]bool{}
	var wg sync.WaitGroup
	wg.Add(2)

	q := New(func(ctx context.Context, job string) {
		mu.Lock()
		ran[job] = true
		mu.Unlock()
		wg.Done()
	})

	q.Enqueue(context.Background(), "key1", "A")
	q.Enqueue(context.Background(), "key2", "B")

	waitWithTimeout(t, &wg, testTimeout)

	mu.Lock()
	defer mu.Unlock()
	if !ran["A"] || !ran["B"] {
		t.Errorf("ran = %v, want both A and B to have run", ran)
	}
}

func TestQueue_SupersededJobsAreCancelledAndCollapsed(t *testing.T) {
	type event struct {
		job       string
		cancelled bool
	}

	started := make(chan string)
	proceed := make(chan struct{})
	doneCh := make(chan event, 10)

	q := New(func(ctx context.Context, job string) {
		started <- job
		select {
		case <-ctx.Done():
			doneCh <- event{job, true}
		case <-proceed:
			doneCh <- event{job, false}
		}
	})

	parent := context.Background()
	q.Enqueue(parent, "k", "A")

	waitForChan(t, started, testTimeout, "job A to start")

	q.Enqueue(parent, "k", "B")
	q.Enqueue(parent, "k", "C")

	ev := waitForEvent(t, doneCh, testTimeout)
	if ev.job != "A" || !ev.cancelled {
		t.Fatalf("first completed job = %+v, want {A, cancelled=true}", ev)
	}

	waitForChan(t, started, testTimeout, "job C to start")
	proceed <- struct{}{}

	ev = waitForEvent(t, doneCh, testTimeout)
	if ev.job != "C" || ev.cancelled {
		t.Fatalf("second completed job = %+v, want {C, cancelled=false}", ev)
	}

	select {
	case extra := <-doneCh:
		t.Fatalf("unexpected extra job ran: %+v", extra)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestQueue_RunsSequentiallyNotConcurrentlyForSameKey(t *testing.T) {
	var mu sync.Mutex
	active := 0
	maxActive := 0

	q := New(func(ctx context.Context, job string) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
	})

	q.Enqueue(context.Background(), "same", "A")
	time.Sleep(2 * time.Millisecond)
	q.Enqueue(context.Background(), "same", "B")
	time.Sleep(2 * time.Millisecond)
	q.Enqueue(context.Background(), "same", "C")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	gotMax := maxActive
	mu.Unlock()
	if gotMax > 1 {
		t.Errorf("max concurrent runs for same key = %d, want <= 1", gotMax)
	}
}

func TestQueue_WorkerRemovedAfterDrain(t *testing.T) {
	done := make(chan struct{})
	q := New(func(ctx context.Context, job string) {
		close(done)
	})

	q.Enqueue(context.Background(), "k", "A")

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for job to run")
	}

	deadline := time.Now().Add(testTimeout)
	for q.Len() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("Len() = %d after drain finished, want 0 (worker map entry leaked)", q.Len())
		}
		time.Sleep(time.Millisecond)
	}
}

func waitWithTimeout(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("timed out waiting for jobs to complete")
	}
}

func waitForChan(t *testing.T, ch chan string, timeout time.Duration, what string) string {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for %s", what)
		return ""
	}
}

func waitForEvent[T any](t *testing.T, ch chan T, timeout time.Duration) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(timeout):
		t.Fatal("timed out waiting for event")
		var zero T
		return zero
	}
}
