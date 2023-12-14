package bfrest_test

import (
	"sync"
	"testing"

	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

func TestGetCache(t *testing.T) {
	bfrest.ResetCache()
	// Test with non-zero maxAgeSeconds
	cache := bfrest.GetCache(500)
	if cache.MaxAge != 500 {
		t.Errorf("Expected maxAge to be %v, got %v", 500, cache.MaxAge)
	}

	bfrest.ResetCache()
	// Test with zero maxAgeSeconds
	cache = bfrest.GetCache(0)
	if cache.MaxAge != 300 {
		t.Errorf("Expected maxAge to be %v, got %v", 300, cache.MaxAge)
	}

	bfrest.ResetCache()
}

func TestAddServer(t *testing.T) {
	cache := &bfrest.BigFixCache{
		ServerCache: &sync.Map{},
		MaxAge:      300,
	}

	// Test adding a new server
	_, err := cache.AddServer("http://test.com", "username", "password", 10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that the server was added to the ServerCache
	value, ok := cache.ServerCache.Load("http://test.com")
	if !ok {
		t.Errorf("Server not found in ServerCache")
	}

	serverCache, ok := value.(*bfrest.BigFixServerCache)
	if !ok {
		t.Errorf("Expected value to be of type *BigFixServerCache, got %T", value)
	}

	if serverCache.ServerName != "http://test.com" {
		t.Errorf("Expected ServerName to be %v, got %v", "http://test.com", serverCache.ServerName)
	}

	// Test adding a server that already exists
	_, err = cache.AddServer("http://test.com", "username", "password", 10)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
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
