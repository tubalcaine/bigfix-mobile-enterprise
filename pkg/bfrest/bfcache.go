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

func (cache BigFixCache) Get(url, username, passwd string) (*CacheItem, Error) {
	baseURL := parseBaseURL(url)

	if cacheInstance == nil {
		cacheInstance = &BigFixCache{
			serverCache: make(map[string]*BigFixServerCache),
			maxAge:      300,
		}
	}

	if cache.serverCache[baseURL] == nil {
		cache.serverCache[baseURL] = &BigFixServerCache{
			serverName: baseURL,
			cpool:      bfrest.NewPool(baseURL, username, passwd, 5),
			cacheMap:   make(map[string]CacheItem),
			maxAge:     cache.maxAge,
		}
	}

	var serverCache = cache.serverCache[baseURL]

	// If the result doesn't exist or is too old, pull it from the server
	if serverCache.cacheMap[url] == nil {
		conn := serverCache.cpool.Acquire()
		defer conn.Release()

		rawXML, err := conn.Get(url)
		serverCache.cacheMap[url] = &CacheItem{
			timestamp: time.Now().Unix(),
			rawXML:    rawXML,
		}
	} else if time.Now().Unix()-serverCache.cacheMap[url].timestamp > serverCache.maxAge {
		conn := serverCache.cpool.Acquire()
		defer conn.Release()

		rawXML, err := conn.Get(url)

		serverCache.cacheMap[url] = &CacheItem{
			timestamp: time.Now().Unix(),
			rawXML:    rawXML,
		}
	}

	return &serverCache.cacheMap[url], nil
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
