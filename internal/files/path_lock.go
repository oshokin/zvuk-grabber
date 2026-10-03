package files

import (
	"path/filepath"
	"sync"
)

// pathLock tracks per-path lock state and active holder count.
type pathLock struct {
	// refCount is the number of active holders for this path lock.
	refCount int64
	// ch is a buffered channel used as a per-path mutex.
	ch chan struct{}
}

// PathLocks serializes access to the same filesystem path.
type PathLocks struct {
	// mu protects the locks map and reference counts.
	mu sync.Mutex
	// locks maps cleaned paths to per-path lock state.
	locks map[string]*pathLock
}

// NewPathLocks creates an empty path lock registry.
func NewPathLocks() *PathLocks {
	return &PathLocks{
		locks: make(map[string]*pathLock),
	}
}

// Lock acquires a mutex for the given path and returns an unlock function.
func (l *PathLocks) Lock(path string) func() {
	if l == nil {
		return func() {}
	}

	cleanPath := filepath.Clean(path)
	if cleanPath == "" || cleanPath == "." {
		return func() {}
	}

	l.mu.Lock()

	lock, exists := l.locks[cleanPath]
	if !exists {
		lock = &pathLock{
			ch: make(chan struct{}, 1),
		}
		lock.ch <- struct{}{}

		l.locks[cleanPath] = lock
	}

	lock.refCount++
	l.mu.Unlock()

	<-lock.ch

	return func() {
		lock.ch <- struct{}{}

		l.mu.Lock()

		lock.refCount--
		if lock.refCount == 0 {
			delete(l.locks, cleanPath)
		}

		l.mu.Unlock()
	}
}
