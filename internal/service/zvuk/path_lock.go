package zvuk

import "path/filepath"

// pathLock serializes concurrent access to the same filesystem path.
type pathLock struct {
	// refCount tracks how many callers currently hold the lock.
	refCount int64
	// ch is a buffered channel used as a mutex for the path.
	ch chan struct{}
}

// lockPath acquires an exclusive lock for the given path and returns an unlock function.
func (s *ServiceImpl) lockPath(path string) func() {
	cleanPath := filepath.Clean(path)
	if cleanPath == "" || cleanPath == "." {
		return func() {}
	}

	s.filePathLocksMutex.Lock()
	if s.filePathLocks == nil {
		s.filePathLocks = make(map[string]*pathLock)
	}

	lock, exists := s.filePathLocks[cleanPath]
	if !exists {
		lock = &pathLock{
			ch: make(chan struct{}, 1),
		}
		lock.ch <- struct{}{}

		s.filePathLocks[cleanPath] = lock
	}

	lock.refCount++
	s.filePathLocksMutex.Unlock()

	<-lock.ch

	return func() {
		lock.ch <- struct{}{}

		s.filePathLocksMutex.Lock()

		lock.refCount--
		if lock.refCount == 0 {
			delete(s.filePathLocks, cleanPath)
		}

		s.filePathLocksMutex.Unlock()
	}
}
