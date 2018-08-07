package internal

import (
	"runtime"
	"sync/atomic"
)

const (
	PollEvent_None  = 0
	PollEvent_Read  = 1
	PollEvent_Write = 2
)

type AddConnection struct {
	FD int
}

// this is a good candiate for a lock-free structure.

type Spinlock struct{ lock uintptr }

func (l *Spinlock) Lock() {
	for !atomic.CompareAndSwapUintptr(&l.lock, 0, 1) {
		runtime.Gosched()
	}
}
func (l *Spinlock) Unlock() {
	atomic.StoreUintptr(&l.lock, 0)
}

type noteQueue struct {
	mu    Spinlock
	notes []interface{}
	n     int64
}

func (q *noteQueue) Add(note interface{}) (one bool) {
	q.mu.Lock()
	n := atomic.AddInt64(&q.n, 1)
	q.notes = append(q.notes, note)
	q.mu.Unlock()
	return n == 1
}

func (q *noteQueue) ForEach(iter func(note interface{}) error) error {
	if atomic.LoadInt64(&q.n) == 0 {
		return nil
	}
	q.mu.Lock()
	if len(q.notes) == 0 {
		q.mu.Unlock()
		return nil
	}
	notes := q.notes
	atomic.StoreInt64(&q.n, 0)
	q.notes = nil
	q.mu.Unlock()
	for _, note := range notes {
		if err := iter(note); err != nil {
			return err
		}
	}
	return nil
}
