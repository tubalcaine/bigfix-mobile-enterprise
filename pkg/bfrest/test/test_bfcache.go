package bfrest

import (
	"testing"
)
package bfrest

import (
    "testing"
    "sync"
)

func TestGetCache(t *testing.T) {
	resetCache()
    // Test with non-zero maxAgeSeconds
    cache := GetCache(500)
    if cache.maxAge != 500 {
        t.Errorf("Expected maxAge to be %v, got %v", 500, cache.maxAge)
    }

	resetCache()
    // Test with zero maxAgeSeconds
    cache = GetCache(0)
    if cache.maxAge != 300 {
        t.Errorf("Expected maxAge to be %v, got %v", 300, cache.maxAge)
    }

    // Test that ServerCache is initialized
    if _, ok := cache.ServerCache.(*sync.Map); !ok {
        t.Errorf("Expected ServerCache to be of type *sync.Map, got %T", cache.ServerCache)
    }
	resetCache()
}

func TestServerUser(t *testing.T) {
	// TODO: Write test cases for ServerUser field
}

func TestServerPass(t *testing.T) {
	// TODO: Write test cases for ServerPass field
}

func TestCacheMap(t *testing.T) {
	// TODO: Write test cases for CacheMap field
}

func TestMaxAge(t *testing.T) {
	// TODO: Write test cases for MaxAge field
}

func TestCacheItem(t *testing.T) {
	// TODO: Write test cases for CacheItem struct
}
