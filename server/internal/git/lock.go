package git

import "sync"

// repoLocks stores per-repository write locks to ensure thread safety
// for concurrent write operations (e.g., branch creation, file commits) on the same repo.
var repoLocks = struct {
	sync.RWMutex
	locks map[string]*sync.Mutex
}{
	locks: make(map[string]*sync.Mutex),
}

// GetRepoLock returns the write lock for the given repository.
// Creates a new lock if one does not exist.
func GetRepoLock(repoPath string) *sync.Mutex {
	repoLocks.Lock()
	defer repoLocks.Unlock()

	if lock, exists := repoLocks.locks[repoPath]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	repoLocks.locks[repoPath] = lock
	return lock
}
