package zvuk

import "path/filepath"

type pathLock struct {
	refCount int64
	ch       chan struct{}
}

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
