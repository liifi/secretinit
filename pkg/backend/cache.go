package backend

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
)

var debugEnabled = os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG"

// debugLog prints debug messages to stderr if debugEnabled is true.
func debugLog(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Cache provides a thread-safe in-memory cache for backend data
type Cache struct {
	data  map[string]string
	mutex sync.RWMutex
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	return &Cache{
		data: make(map[string]string),
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	value, exists := c.data[key]
	if exists {
		debugLog("Cache hit for key: %s", hashKey(key))
	} else {
		debugLog("Cache miss for key: %s", hashKey(key))
	}
	return value, exists
}

// Set stores a value in the cache
func (c *Cache) Set(key, value string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = value
	debugLog("Cached value for key: %s", hashKey(key))
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]string)
	debugLog("Cache cleared")
}

// Size returns the number of cached entries
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.data)
}

// hashKey returns a hash of the key for debug logging (to avoid exposing sensitive data)
func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h)[:8] // First 8 chars for readability
}

// globalCache is a shared cache instance for all backends
var globalCache = NewCache()

// GetGlobalCache returns the global cache instance
func GetGlobalCache() *Cache {
	return globalCache
}

// ClearGlobalCache clears the global cache
func ClearGlobalCache() {
	globalCache.Clear()
}

// GetGlobalCacheSize returns the size of the global cache
func GetGlobalCacheSize() int {
	return globalCache.Size()
}
