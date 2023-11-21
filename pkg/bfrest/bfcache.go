package bfrest

import (
	"encoding/json"
	"encoding/xml"
	"net/url"
	"strings"
	"time"
)

type CacheItem struct {
	Timestamp int64
	RawXML    string
	Json      string
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

func getBaseUrl(fullURL string) string {
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return ""
	}

	scheme := parsedURL.Scheme
	host := parsedURL.Host
	port := ""

	if strings.Contains(host, ":") {
		hostPort := strings.Split(host, ":")
		host = hostPort[0]
		port = hostPort[1]
	}

	return scheme + "://" + host + ":" + port
}

func Get(url, username, passwd string) (*CacheItem, error) {
	baseURL := getBaseUrl(url)

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
	if serverCache.cacheMap[url] == nil || (time.Now().Unix()-serverCache.cacheMap[url].Timestamp) > int64(serverCache.maxAge) {
		conn, err := serverCache.cpool.Acquire()

		if err != nil {
			return nil, err
		}

		defer serverCache.cpool.Release(conn)

		rawXML, err := conn.Get(url)

		if err != nil {
			return nil, err
		}

		var besapi BESAPI
		var bes BES
		var jsonValue []byte

		if strings.Contains(rawXML, "BESAPI") {
			err = xml.Unmarshal(([]byte)(rawXML), &besapi)
			if err != nil {
				return nil, err
			}

			jsonValue, err = json.Marshal(&besapi)
			if err != nil {
				return nil, err
			}
		} else {
			err = xml.Unmarshal(([]byte)(rawXML), &bes)
			if err != nil {
				return nil, err
			}jsonValue

			jsonValue, err = json.Marshal(&bes)
			if err != nil {
				return nil, err
			}
		}

		jStr := (string)(jsonValue)

		serverCache.cacheMap[url] = &CacheItem{
			Timestamp: time.Now().Unix(),
			RawXML:    rawXML,
			Json:      jStr,
		}
	}

	return serverCache.cacheMap[url], nil
}
