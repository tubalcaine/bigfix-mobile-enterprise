package bfrest

import (
	"strings"
	"time"
)

type CacheItem struct {
	timestamp int64
	rawXML    string
}

type BigFixServerCache struct {
	serverName string
	cpool      *Pool
	cacheMap   map[string]*CacheItem
	maxAge     uint64
}

type BigFixCache struct {
	serverCache map[string]*BigFixServerCache
	maxAge      uint64
}

var cacheInstance *BigFixCache

func GetCacheInstance() *BigFixCache {
	if cacheInstance == nil {
		cacheInstance = &BigFixCache{
			serverCache: make(map[string]*BigFixServerCache),
			maxAge:      300,
		}
	}
	return cacheInstance
}

func Get(url, username, passwd string) (*CacheItem, error) {
	baseURL := parseBaseURL(url)
	cache := GetCacheInstance()

	if cache.serverCache[baseURL] == nil {
		newpool, _ := NewPool(baseURL, username, passwd, 5)

		cache.serverCache[baseURL] = &BigFixServerCache{
			serverName: baseURL,
			cpool:      newpool,
			cacheMap:   make(map[string]*CacheItem),
			maxAge:     cache.maxAge,
		}
	}

	var serverCache = cache.serverCache[baseURL]

	// If the result doesn't exist or is too old, pull it from the server
	if serverCache.cacheMap[url] == nil {
		conn, err := serverCache.cpool.Acquire()

		if err != nil {
			return nil, err
		}

		defer serverCache.cpool.Release(conn)

		rawXML, err := conn.Get(url)

		if err != nil {
			return nil, err
		}

		serverCache.cacheMap[url] = &CacheItem{
			timestamp: time.Now().Unix(),
			rawXML:    rawXML,
		}
	}

	return serverCache.cacheMap[url], nil
}

func parseBaseURL(url string) string {
	// Find the index of the first occurrence of ":"
	colonIndex := strings.Index(url, ":")
	// Find the index of the first occurrence of "/"
	slashIndex := strings.Index(url, "/")

	// Extract the substring from the start to the port
	baseURL := url[colonIndex+3 : slashIndex]

	return baseURL
}
