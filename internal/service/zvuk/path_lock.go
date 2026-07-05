package zvuk

// lockPath acquires an exclusive lock for the given path and returns an unlock function.
func (s *ServiceImpl) lockPath(path string) func() {
	if s == nil || s.pathLocks == nil {
		return func() {}
	}

	return s.pathLocks.Lock(path)
}
