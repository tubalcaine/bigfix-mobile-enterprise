package bfrest

// The package bfrest provides a cache implementation for BigFix servers and their data.
// It includes functionality to add servers to the cache, retrieve data from the cache,
// and populate the cache with commonly accessed data.

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

// BigFixCache is a cache of BigFix servers and their data.
// It is a singleton that is accessed by multiple goroutines.
// It contains a map of BigFixServerCache instances.
type BigFixCache struct {
	ServerCache *sync.Map
	MaxAge      uint64
}

// BigFixServerCache represents a cache for storing one BigFix
// server's information. It contains a map of CacheItems.
type BigFixServerCache struct {
	ServerName string
	ServerUser string
	ServerPass string
	cpool      *Pool
	CacheMap   *sync.Map
	MaxAge     uint64
}

// CacheItem represents the result of a single BigFix GET result
// from a single BigFix server. This is stored in the CacheMap of
// a BigFixServerCache.
// Timestamp represents the time when the cache item was created.
// RawXML stores the raw XML data associated with the cache item.
// -- In the future we may discard this data after unmarshalling it.
// Json stores the JSON representation of the cache item.
type CacheItem struct {
	Timestamp int64
	RawXML    string
	Json      string
}

var cacheInstance *BigFixCache
var cacheMu = &sync.Mutex{}

// GetCache is a singleton cache constructor
func GetCache(maxAgeSeconds uint64) *BigFixCache {
	cacheMu.Lock()
	if maxAgeSeconds == 0 {
		maxAgeSeconds = 300
	}

	defer cacheMu.Unlock()
	if cacheInstance == nil {
		cacheInstance = &BigFixCache{
			ServerCache: &sync.Map{},
			MaxAge:      maxAgeSeconds,
		}
	}

	return cacheInstance
}

// ResetCache should only be called for testing purposes. There is
// no reason to reset the cache in production.
func ResetCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheInstance = nil
}

// AddServer adds a BigFix server to the cache.
// It creates a new cache instance for the server if it doesn't already exist.
// The server is identified by its URL, username, and password.
// The poolSize parameter specifies the maximum number of connections in the connection pool.
// Returns the updated BigFixCache instance and an error if the server cache already exists.
func (cache *BigFixCache) AddServer(url, username, passwd string, poolSize int) (*BigFixCache, error) {
	baseURL := getBaseUrl(url)

	fmt.Fprintf(os.Stderr, "Get URL: %s\n", url)

	_, err := cache.ServerCache.Load(baseURL)

	// If the BigFixServerCache is not found...
	if !err {
		newpool, _ := NewPool(baseURL, username, passwd, poolSize)

		scInstance := &BigFixServerCache{
			ServerName: baseURL,
			cpool:      newpool,
			MaxAge:     cache.MaxAge,
			CacheMap:   &sync.Map{},
		}

		cache.ServerCache.Store(baseURL, scInstance)
		// Reload scValue with the newly created cache
		_, _ = cache.ServerCache.Load(baseURL)
		return cache, nil
	}

	return nil, fmt.Errorf("server cache %s already exists", baseURL)
}

// getBaseUrl returns the base URL extracted from the given full URL.
// It parses the full URL and extracts the scheme, host, and port (if present).
// The base URL is then constructed by combining the scheme, host, and port.
// If the full URL is invalid, an empty string is returned.
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

// silentGet is a wrapper around Get that does not print to stderr.
// It is intended to be called as a goroutine to load the cache
// with the most commonly accessed data in the background. It ignores
// errors.
func (cache *BigFixCache) silentGet(url string) {
	res, err := cache.Get(url)

	if res == nil || err != nil {
		fmt.Fprintf(os.Stderr, "For URL: %s\n", url)
		fmt.Fprintln(os.Stderr, res)
		fmt.Fprintf(os.Stderr, "Silent GET failed: %s\n", err)
		//		os.Exit(1)
	}
}

// Get retrieves a cache item for the given URL from the BigFixCache.
// If the cache item is found and not expired, it is returned as a *CacheItem.
// If the cache item is not found or expired, it is retrieved from the server
// using the retrieveBigFixData function and stored in the cache before being returned.
// If the server cache does not exist for the given URL, an error is returned.
func (cache *BigFixCache) Get(url string) (*CacheItem, error) {
	baseURL := getBaseUrl(url)

	scValue, ok := cache.ServerCache.Load(baseURL)

	// If the BigFixServerCache is not found...
	if !ok {
		return nil, fmt.Errorf("server cache does not exist for %s", baseURL)
	}

	// Make the type assertion and handle failure
	sc, _ := scValue.(*BigFixServerCache)

	// We now have the server's cache. Check to see if we have the
	// requested URL and if it is not expired

	// If the result doesn't exist or is too old, pull it from the server
	value, ok := sc.CacheMap.Load(url)

	var cm *CacheItem

	if !ok {
		// Cache miss

		cm, err := retrieveBigFixData(url, sc)
		if err != nil {
			return nil, err
		}
		sc.CacheMap.Store(url, cm)
		return cm, nil
	}

	cm, ok = value.(*CacheItem)

	if !ok {
		return nil, fmt.Errorf("type failure loading cache item for %s", url)
	}

	if time.Now().Unix()-cm.Timestamp > int64(sc.MaxAge) {
		// Cache expired
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

// retrieveBigFixData retrieves BigFix data from the specified URL and returns a CacheItem containing the raw XML and JSON representation of the data.
// It acquires a connection from the BigFixServerCache connection pool, makes a GET request to the URL, and unmarshals the XML response into either a BESAPI or BES struct.
// The JSON representation of the struct is then marshaled and returned as part of the CacheItem.
// If any errors occur during the process, the acquired connection is released and the error is returned.
func retrieveBigFixData(urlStr string, sc *BigFixServerCache) (*CacheItem, error) {
	conn, err := sc.cpool.Acquire()

	if err != nil {
		fmt.Printf("For URL %s\nError acquiring connection: %s\n\n", urlStr, err)
		return nil, err
	}

	rawResponse, err := conn.Get(urlStr)

	if err != nil {
		sc.cpool.Release(conn)
		return nil, err
	}

	// Check if this is an /api/query endpoint with JSON output format
	parsedURL, parseErr := url.Parse(urlStr)
	if parseErr == nil && strings.Contains(parsedURL.Path, "/api/query") {
		queryParams, queryErr := url.ParseQuery(parsedURL.RawQuery)
		if queryErr == nil {
			// Check for output=json or format=json parameters
			outputFormat := queryParams.Get("output")
			formatParam := queryParams.Get("format")
			
			if outputFormat == "json" || formatParam == "json" {
				// For JSON format requests, pass through the JSON response directly
				sc.cpool.Release(conn)
				return &CacheItem{
					Timestamp: time.Now().Unix(),
					RawXML:    rawResponse, // Still called RawXML for compatibility, but contains JSON
					Json:      rawResponse,
				}, nil
			}
		}
	}

	// Default behavior: parse XML and convert to JSON
	var besapi BESAPI
	var bes BES
	var jsonValue []byte

	if strings.Contains(rawResponse, "BESAPI") {
		err = xml.Unmarshal(([]byte)(rawResponse), &besapi)
		if err != nil {
			sc.cpool.Release(conn)
fmt.Printf("DEBUG.BESAPI: for url [%s]\nxml.Unmarshal failed, err [%s]\nRaw result [%s]\n------------\n\n", urlStr, err, rawResponse)
			return nil, err
		}

		jsonValue, err = json.Marshal(&besapi)
		if err != nil {
fmt.Printf("DEBUG.BESAPI: for url [%s]\njson.Marshal failed, err [%s]\nRaw json [%s]\n------------\n\n", urlStr, err, jsonValue)
			sc.cpool.Release(conn)
			return nil, err
		}
	} else {
		err = xml.Unmarshal(([]byte)(rawResponse), &bes)
		if err != nil {
fmt.Printf("DEBUG.BES: for url [%s]\nxml.Unmarshal failed, err [%s]\nRaw result [%s]\n------------\n\n", urlStr, err, rawResponse)
			sc.cpool.Release(conn)
			return nil, err
		}

		jsonValue, err = json.Marshal(&bes)
		if err != nil {
fmt.Printf("DEBUG.BES: for url [%s]\njson.Marshal failed, err [%s]\nRaw json [%s]\n------------\n\n", urlStr, err, jsonValue)
			sc.cpool.Release(conn)
			return nil, err
		}
	}

	jStr := string(jsonValue)

	sc.cpool.Release(conn)
	return &CacheItem{
		Timestamp: time.Now().Unix(),
		RawXML:    rawResponse,
		Json:      jStr,
	}, nil
}

// PopulateCoreTypes populates the BigFixCache with core types by making API calls to the specified serverUrl.
// It retrieves actions, computers, sites, and their corresponding resources and content.
// The maxAgeSeconds parameter specifies the maximum age in seconds for the cached data.
// This method runs asynchronously, making concurrent API calls to improve performance.
// It returns an error if any API call fails.
// This method really isn't necessary, but it is useful for populating the cache with commonly accessed data.
func (cache *BigFixCache) PopulateCoreTypes(serverUrl string, maxAgeSeconds uint64) error {
	var besapi BESAPI

	result, err := cache.Get(serverUrl + "/api/actions")
	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, action := range besapi.Action {
		//		silentGet(action.Resource, username, password)
		go cache.silentGet(action.Resource)
		go cache.silentGet(action.Resource + "/status")
	}

	result, err = cache.Get(serverUrl + "/api/computers")
	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, computer := range besapi.Computer {
		//		silentGet(computer.Resource, username, password)
		go cache.silentGet(computer.Resource)
	}

	result, err = cache.Get(serverUrl + "/api/sites")

	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, site := range besapi.CustomSite {
		//		silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource)
		go cache.silentGet(site.Resource + "/content")
	}

	for _, site := range besapi.ExternalSite {
		//		silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource)
		go cache.silentGet(site.Resource + "/content")
	}

	for _, site := range besapi.OperatorSite {
		//		silentGet(site.Resource, username, password)
		go cache.silentGet(site.Resource)
		go cache.silentGet(site.Resource + "/content")
	}

	if besapi.ActionSite != nil {
		go cache.silentGet(besapi.ActionSite.Resource)
		go cache.silentGet(besapi.ActionSite.Resource + "/content")
	}

	return nil
}
