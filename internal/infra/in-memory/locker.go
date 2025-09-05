package inmemory

import (
	"context"
	"sync"
)

type InMemoryLocker struct {
	mu    sync.Mutex
	locks map[string]struct{}
}

func NewInMemoryLocker() *InMemoryLocker {
	return &InMemoryLocker{
		locks: make(map[string]struct{}),
	}
}

func (l *InMemoryLocker) Lock(_ context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.locks[key] = struct{}{}
	return nil
}

func (l *InMemoryLocker) Unlock(_ context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.locks, key)
	return nil
}
