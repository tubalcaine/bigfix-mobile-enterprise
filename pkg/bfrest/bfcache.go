package bfrest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type BigFixCache struct {
	ServerCache *sync.Map
	maxAge      uint64
}

type BigFixServerCache struct {
	ServerName string
	cpool      *Pool
	CacheMap   *sync.Map
	MaxAge     uint64
}

type CacheItem struct {
	Timestamp int64
	RawXML    string
	Json      string
}

var cacheInstance *BigFixCache
var cacheMu = &sync.Mutex{}

// Singleton cache constructor
func GetCache(maxAgeSeconds uint64) *BigFixCache {
	cacheMu.Lock()
	if maxAgeSeconds == 0 {
		maxAgeSeconds = 300
	}

	defer cacheMu.Unlock()
	if cacheInstance == nil {
		cacheInstance = &BigFixCache{
			ServerCache: &sync.Map{},
			maxAge:      maxAgeSeconds,
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

func (cache *BigFixCache) silentGet(url, username, passwd string) {
	fmt.Fprintf(os.Stderr, "Silent GET URL: %s\n", url)
	res, err := cache.Get(url, username, passwd)

	if err != nil {
		fmt.Fprintf(os.Stderr, "For URL: %s\n", url)
		fmt.Fprintln(os.Stderr, res)
		fmt.Fprintf(os.Stderr, "Silent GET failed: %s\n", err)
		//		os.Exit(1)
	}
}

func (cache *BigFixCache) Get(url, username, passwd string) (*CacheItem, error) {
	baseURL := getBaseUrl(url)

	fmt.Fprintf(os.Stderr, "Get URL: %s\n", url)

	scValue, err := cache.ServerCache.Load(baseURL)

	// If the BigFixServerCache is not found...
	if !err {
		newpool, _ := NewPool(baseURL, username, passwd, 8)

		scInstance := &BigFixServerCache{
			ServerName: baseURL,
			cpool:      newpool,
			MaxAge:     cache.maxAge,
			CacheMap:   &sync.Map{},
		}

		cache.ServerCache.Store(baseURL, scInstance)
		// Reload scValue with the newly created cache
		scValue, _ = cache.ServerCache.Load(baseURL)
	}

	// Make the type assertion and handle failureserenity:1
	sc, _ := scValue.(*BigFixServerCache)

	// We now have the server's cache. Check to see if we have the
	// requested URL and if it is not expired

	// If the result doesn't exist or is too old, pull it from the server
	value, err := sc.CacheMap.Load(url)

	var cm *CacheItem

	// Cache miss		cache := bfrest.GetCache()

	if !err {
		cm, err := retrieveBigFixData(url, sc)
		if err != nil {
			return nil, err
		}
		sc.CacheMap.Store(url, cm)
		return cm, nil
	}

	cm, err = value.(*CacheItem)

	if !err {
		return nil, fmt.Errorf("type failure loading cache item for %s", url)
	}

	// Cache expired
	if time.Now().Unix()-cm.Timestamp > int64(sc.MaxAge) {
		cm, err := retrieveBigFixData(url, sc)
		if err != nil {
			return nil, err
		}
		sc.CacheMap.Store(url, cm)
		return cm, nil
	}

	// Cache hit
	return cm, nil
}

func retrieveBigFixData(url string, sc *BigFixServerCache) (*CacheItem, error) {
	conn, err := sc.cpool.Acquire()

	if err != nil {
		fmt.Printf("For URL %s\nError acquiring connection: %s\n\n", url, err)
		return nil, err
	}

	rawXML, err := conn.Get(url)

	if err != nil {
		sc.cpool.Release(conn)
		return nil, err
	}

	var besapi BESAPI
	var bes BES
	var jsonValue []byte

	if strings.Contains(rawXML, "BESAPI") {
		err = xml.Unmarshal(([]byte)(rawXML), &besapi)
		if err != nil {
			sc.cpool.Release(conn)
			return nil, err
		}

		jsonValue, err = json.Marshal(&besapi)
		if err != nil {
			sc.cpool.Release(conn)
			return nil, err
		}
	} else {
		err = xml.Unmarshal(([]byte)(rawXML), &bes)
		if err != nil {
			sc.cpool.Release(conn)
			return nil, err
		}

		jsonValue, err = json.Marshal(&bes)
		if err != nil {
			sc.cpool.Release(conn)
			return nil, err
		}
	}

	jStr := string(jsonValue)

	sc.cpool.Release(conn)
	return &CacheItem{
		Timestamp: time.Now().Unix(),
		RawXML:    rawXML,
		Json:      jStr,
	}, nil
}

func PopulateCoreTypes(serverUrl string, username string, password string, maxAgeSeconds uint64) error {
	var besapi BESAPI

	cache := GetCache(maxAgeSeconds)

	result, err := cache.Get(serverUrl+"/api/actions", username, password)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, action := range besapi.Action {
		//		silentGet(action.Resource, username, password)
		go cache.silentGet(action.Resource, username, password)
		go cache.silentGet(action.Resource+"/status", username, password)
	}

	result, err = cache.Get(serverUrl+"/api/computers", username, password)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, computer := range besapi.Computer {
		//		silentGet(computer.Resource, username, password)
		go cache.silentGet(computer.Resource, username, password)
	}

	result, err = cache.Get(serverUrl+"/api/sites", username, password)

	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, site := range besapi.CustomSite {
		//		silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource+"/content", username, password)
	}

	for _, site := range besapi.ExternalSite {
		//		silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource+"/content", username, password)
	}

	for _, site := range besapi.OperatorSite {
		//		silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource+"/content", username, password)
	}

	go cache.silentGet(besapi.ActionSite.Resource, username, password)
	go cache.silentGet(besapi.ActionSite.Resource+"/content", username, password)

	return nil
}
