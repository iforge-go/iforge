package git

import (
	"container/list"
	"strconv"
	"sync"
	"time"
)

// CacheEntry represents a cache entry with expiration
type CacheEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// LRUCache implements a simple LRU cache with TTL
type LRUCache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
	ttl      time.Duration
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
		ttl:      ttl,
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*CacheEntry)

		// Check if expired — remove stale entry
		if time.Now().After(entry.expiresAt) {
			c.removeElement(elem)
			return nil, false
		}

		// Move to front (most recently used)
		c.order.MoveToFront(elem)
		return entry.value, true
	}

	return nil, false
}

// Set adds or updates a value in the cache
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*CacheEntry).value = value
		elem.Value.(*CacheEntry).expiresAt = time.Now().Add(c.ttl)
		return
	}

	// Add new
	entry := &CacheEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}

	elem := c.order.PushFront(entry)
	c.items[key] = elem

	// Evict if over capacity
	if c.order.Len() > c.capacity {
		c.evict()
	}
}

// Delete removes a value from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
}

// DeleteByPrefix removes all entries whose key starts with the given prefix
func (c *LRUCache) DeleteByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, elem := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.removeElement(elem)
		}
	}
}

// Size returns the number of items in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

func (c *LRUCache) evict() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *LRUCache) removeElement(elem *list.Element) {
	c.order.Remove(elem)
	entry := elem.Value.(*CacheEntry)
	delete(c.items, entry.key)
}

// Global cache instances
var (
	// Repository cache - stores repository metadata
	RepositoryCache = NewLRUCache(1000, 5*time.Minute)

	// File content cache - stores file contents
	FileContentCache = NewLRUCache(500, 2*time.Minute)

	// Branch list cache - stores branch lists
	BranchListCache = NewLRUCache(500, 3*time.Minute)

	// Commit cache - stores commit objects
	CommitCache = NewLRUCache(2000, 10*time.Minute)

	// File list cache - stores file lists (5 minutes)
	FileListCache = NewLRUCache(500, 5*time.Minute)

	// Commit list cache - stores commit lists (2 minutes)
	CommitListCache = NewLRUCache(200, 2*time.Minute)
)

// Cache key generators
func RepositoryCacheKey(owner, repo string) string {
	return "repo:" + owner + "/" + repo
}

func FileContentCacheKey(owner, repo, ref, path string) string {
	return "file:" + owner + "/" + repo + ":" + ref + ":" + path
}

func BranchListCacheKey(owner, repo string) string {
	return "branches:" + owner + "/" + repo
}

func CommitCacheKey(owner, repo, hash string) string {
	return "commit:" + owner + "/" + repo + ":" + hash
}

func FileListCacheKey(owner, repo, ref, path string) string {
	return "filelist:" + owner + "/" + repo + ":" + ref + ":" + path
}

func CommitListCacheKey(owner, repo, ref string, limit int) string {
	return "commitlist:" + owner + "/" + repo + ":" + ref + ":" + strconv.Itoa(limit)
}

// RepoCache is the interface that all repo-related caches should implement
// for participation in the CacheManager's centralized invalidation.
type RepoCache interface {
	// InvalidateRepo removes all cache entries for the given repository
	InvalidateRepo(owner, repo string)
}

// CacheManager manages all registered caches and provides centralized invalidation.
// Write operations call InvalidateRepo once, and all registered caches clean up
// their own entries for that repository — no need for callers to know which caches exist.
type CacheManager struct {
	caches []RepoCache
}

var cacheManager = &CacheManager{}

// RegisterCache registers a cache with the CacheManager
func RegisterCache(c RepoCache) {
	cacheManager.caches = append(cacheManager.caches, c)
}

// InvalidateRepo invalidates all registered caches for a specific repository
func InvalidateRepoCaches(owner, repo string) {
	for _, c := range cacheManager.caches {
		c.InvalidateRepo(owner, repo)
	}
}

// InvalidateAllCaches clears all registered caches entirely
func InvalidateAllCaches() {
	RepositoryCache.Clear()
	FileContentCache.Clear()
	BranchListCache.Clear()
	CommitCache.Clear()
	FileListCache.Clear()
	CommitListCache.Clear()
}

// --- Concrete cache implementations ---

// repositoryCacheImpl wraps RepositoryCache for repo-scoped invalidation
type repositoryCacheImpl struct{ c *LRUCache }

func (r repositoryCacheImpl) InvalidateRepo(owner, repo string) {
	r.c.Delete(RepositoryCacheKey(owner, repo))
}

// fileContentCacheImpl wraps FileContentCache for repo-scoped invalidation
type fileContentCacheImpl struct{ c *LRUCache }

func (f fileContentCacheImpl) InvalidateRepo(owner, repo string) {
	f.c.DeleteByPrefix("file:" + owner + "/" + repo + ":")
}

// branchListCacheImpl wraps BranchListCache for repo-scoped invalidation
type branchListCacheImpl struct{ c *LRUCache }

func (b branchListCacheImpl) InvalidateRepo(owner, repo string) {
	b.c.Delete(BranchListCacheKey(owner, repo))
}

// commitCacheImpl wraps CommitCache for repo-scoped invalidation
type commitCacheImpl struct{ c *LRUCache }

func (c commitCacheImpl) InvalidateRepo(owner, repo string) {
	c.c.DeleteByPrefix("commit:" + owner + "/" + repo + ":")
}

// fileListCacheImpl wraps FileListCache for repo-scoped invalidation
type fileListCacheImpl struct{ c *LRUCache }

func (f fileListCacheImpl) InvalidateRepo(owner, repo string) {
	f.c.DeleteByPrefix("filelist:" + owner + "/" + repo + ":")
}

// commitListCacheImpl wraps CommitListCache for repo-scoped invalidation
type commitListCacheImpl struct{ c *LRUCache }

func (c commitListCacheImpl) InvalidateRepo(owner, repo string) {
	c.c.DeleteByPrefix("commitlist:" + owner + "/" + repo + ":")
}

// init registers all caches with the CacheManager
func init() {
	RegisterCache(repositoryCacheImpl{RepositoryCache})
	RegisterCache(fileContentCacheImpl{FileContentCache})
	RegisterCache(branchListCacheImpl{BranchListCache})
	RegisterCache(commitCacheImpl{CommitCache})
	RegisterCache(fileListCacheImpl{FileListCache})
	RegisterCache(commitListCacheImpl{CommitListCache})
}

// InvalidateRepositoryCache invalidates all cache entries for a repository (legacy, kept for backward compatibility)
func InvalidateRepositoryCache(owner, repo string) {
	InvalidateRepoCaches(owner, repo)
}
