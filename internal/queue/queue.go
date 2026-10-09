package queue

import (
	"context"
	"sync"
)

type Runner[T any] func(ctx context.Context, job T)

type Queue[T any] struct {
	run Runner[T]

	mu      sync.Mutex
	workers map[string]*worker[T]
}

type worker[T any] struct {
	pending    T
	hasPending bool
	cancel     context.CancelFunc
	running    bool
}

func New[T any](run Runner[T]) *Queue[T] {
	return &Queue[T]{run: run, workers: map[string]*worker[T]{}}
}

func (q *Queue[T]) Enqueue(parent context.Context, key string, job T) {
	q.mu.Lock()
	w, ok := q.workers[key]
	if !ok {
		w = &worker[T]{}
		q.workers[key] = w
	}
	if w.cancel != nil {
		w.cancel()
	}
	w.pending = job
	w.hasPending = true
	start := !w.running
	if start {
		w.running = true
	}
	q.mu.Unlock()

	if start {
		go q.drain(parent, key, w)
	}
}

func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.workers)
}

func (q *Queue[T]) drain(parent context.Context, key string, w *worker[T]) {
	for {
		q.mu.Lock()
		if !w.hasPending {
			w.running = false
			if cur, ok := q.workers[key]; ok && cur == w {
				delete(q.workers, key)
			}
			q.mu.Unlock()
			return
		}
		job := w.pending
		w.hasPending = false
		jobCtx, cancel := context.WithCancel(parent)
		w.cancel = cancel
		q.mu.Unlock()

		q.run(jobCtx, job)
		cancel()

		q.mu.Lock()
		w.cancel = nil
		q.mu.Unlock()
	}
}
