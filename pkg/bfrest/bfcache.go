package bfrest

// The package bfrest provides a cache implementation for BigFix servers and their data.
// It includes functionality to add servers to the cache, retrieve data from the cache,
// and populate the cache with commonly accessed data.

import (
	"crypto/md5"
	"encoding/hex"
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
	ServerCache      *sync.Map
	MaxAge           uint64
	MaxCacheLifetime uint64 // Maximum lifetime for any cache item in seconds
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
// Json stores the JSON representation of the cache item.
// MaxAge stores the cache expiration time in seconds for this item (can grow dynamically).
// BaseMaxAge stores the original MaxAge value from the server for increment calculations.
// ContentHash stores the MD5 hash of the raw data from the server.
type CacheItem struct {
	Timestamp   int64
	Json        string
	MaxAge      uint64
	BaseMaxAge  uint64
	ContentHash string
}

var cacheInstance *BigFixCache
var cacheMu = &sync.Mutex{}

// GetCache is a singleton cache constructor
func GetCache(maxAgeSeconds uint64, maxCacheLifetime uint64) *BigFixCache {
	cacheMu.Lock()
	if maxAgeSeconds == 0 {
		maxAgeSeconds = 300
	}
	if maxCacheLifetime == 0 {
		maxCacheLifetime = 86400 // Default to 24 hours
	}

	defer cacheMu.Unlock()
	if cacheInstance == nil {
		cacheInstance = &BigFixCache{
			ServerCache:      &sync.Map{},
			MaxAge:           maxAgeSeconds,
			MaxCacheLifetime: maxCacheLifetime,
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
// The maxAge parameter specifies the cache expiration time in seconds for this server.
// Returns the updated BigFixCache instance and an error if the server cache already exists.
func (cache *BigFixCache) AddServer(url, username, passwd string, poolSize int, maxAge uint64) (*BigFixCache, error) {
	baseURL := getBaseUrl(url)

	fmt.Fprintf(os.Stderr, "Get URL: %s\n", url)

	_, err := cache.ServerCache.Load(baseURL)

	// If the BigFixServerCache is not found...
	if !err {
		newpool, _ := NewPool(baseURL, username, passwd, poolSize)

		// Use server-specific maxAge, or fall back to cache default if not specified
		serverMaxAge := maxAge
		if serverMaxAge == 0 {
			serverMaxAge = cache.MaxAge
		}

		scInstance := &BigFixServerCache{
			ServerName: baseURL,
			cpool:      newpool,
			MaxAge:     serverMaxAge,
			CacheMap:   &sync.Map{},
		}

		fmt.Fprintf(os.Stderr, "Added server %s with MaxAge: %d seconds\n", baseURL, serverMaxAge)

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
// If the cache item is not found, expired, or has empty Json, it is retrieved from the server.
// When fetching fresh data, the MD5 hash is compared to detect changes:
// - If unchanged: MaxAge is extended (up to MaxCacheLifetime) for efficient caching of stable content
// - If changed: MaxAge resets to BaseMaxAge to ensure fresh content is refreshed regularly
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
		// Cache miss - first time accessing this URL

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

	// Check if cache item needs refresh: Json is empty (cleared by GC) or expired
	isEmpty := cm.Json == ""
	isExpired := time.Now().Unix()-cm.Timestamp > int64(cm.MaxAge)
	needsRefresh := isEmpty || isExpired

	fmt.Fprintf(os.Stderr, "\n=== CACHE CHECK for %s ===\n", url)
	fmt.Fprintf(os.Stderr, "  Current state: isEmpty=%v, isExpired=%v, needsRefresh=%v\n", isEmpty, isExpired, needsRefresh)
	fmt.Fprintf(os.Stderr, "  Current values: Timestamp=%d, MaxAge=%d, BaseMaxAge=%d, JSON length=%d, Hash=%s\n",
		cm.Timestamp, cm.MaxAge, cm.BaseMaxAge, len(cm.Json), cm.ContentHash[:8])

	if needsRefresh {
		fmt.Fprintf(os.Stderr, "  --> Refreshing from server...\n")

		// Fetch fresh data from server
		newItem, err := retrieveBigFixData(url, sc)
		if err != nil {
			return nil, err
		}

		fmt.Fprintf(os.Stderr, "  Fresh data retrieved: JSON length=%d, Hash=%s\n", len(newItem.Json), newItem.ContentHash[:8])

		// Determine if content has changed by comparing hashes
		hashMatches := cm.ContentHash != "" && newItem.ContentHash == cm.ContentHash
		fmt.Fprintf(os.Stderr, "  Hash comparison: old=%s, new=%s, matches=%v\n",
			cm.ContentHash[:8], newItem.ContentHash[:8], hashMatches)

		var updatedItem *CacheItem

		if hashMatches {
			// Content unchanged - restore Json (if it was cleared) and extend MaxAge
			newMaxAge := cm.MaxAge + cm.BaseMaxAge
			if newMaxAge > cache.MaxCacheLifetime {
				fmt.Fprintf(os.Stderr, "  MaxAge extension capped: would be %d, capping to %d (MaxCacheLifetime)\n",
					newMaxAge, cache.MaxCacheLifetime)
				newMaxAge = cache.MaxCacheLifetime
			}

			fmt.Fprintf(os.Stderr, "  HASH MATCHED - Content unchanged!\n")
			fmt.Fprintf(os.Stderr, "    Extending MaxAge: %d + %d = %d\n", cm.MaxAge, cm.BaseMaxAge, newMaxAge)
			fmt.Fprintf(os.Stderr, "    Restoring JSON: %d bytes\n", len(newItem.Json))

			// Create updated item with extended MaxAge, restored Json, and same content hash
			updatedItem = &CacheItem{
				Timestamp:   time.Now().Unix(),
				Json:        newItem.Json,
				MaxAge:      newMaxAge,
				BaseMaxAge:  cm.BaseMaxAge,
				ContentHash: cm.ContentHash, // Keep old hash since content matches
			}

			fmt.Fprintf(os.Stderr, "  Values to be stored:\n")
			fmt.Fprintf(os.Stderr, "    Timestamp:   %d (now)\n", updatedItem.Timestamp)
			fmt.Fprintf(os.Stderr, "    MaxAge:      %d\n", updatedItem.MaxAge)
			fmt.Fprintf(os.Stderr, "    BaseMaxAge:  %d\n", updatedItem.BaseMaxAge)
			fmt.Fprintf(os.Stderr, "    JSON length: %d\n", len(updatedItem.Json))
			fmt.Fprintf(os.Stderr, "    ContentHash: %s\n", updatedItem.ContentHash[:8])
		} else {
			fmt.Fprintf(os.Stderr, "  HASH CHANGED - Content has changed!\n")
			fmt.Fprintf(os.Stderr, "    Resetting MaxAge to BaseMaxAge: %d\n", cm.BaseMaxAge)
			fmt.Fprintf(os.Stderr, "    Updating hash: %s -> %s\n", cm.ContentHash[:8], newItem.ContentHash[:8])

			// Content changed - store new data with new hash and reset to BaseMaxAge
			updatedItem = &CacheItem{
				Timestamp:   time.Now().Unix(),
				Json:        newItem.Json,
				MaxAge:      cm.BaseMaxAge, // Reset to base, not newItem.MaxAge
				BaseMaxAge:  cm.BaseMaxAge,
				ContentHash: newItem.ContentHash, // Update to new hash
			}

			fmt.Fprintf(os.Stderr, "  Values to be stored:\n")
			fmt.Fprintf(os.Stderr, "    Timestamp:   %d (now)\n", updatedItem.Timestamp)
			fmt.Fprintf(os.Stderr, "    MaxAge:      %d\n", updatedItem.MaxAge)
			fmt.Fprintf(os.Stderr, "    BaseMaxAge:  %d\n", updatedItem.BaseMaxAge)
			fmt.Fprintf(os.Stderr, "    JSON length: %d\n", len(updatedItem.Json))
			fmt.Fprintf(os.Stderr, "    ContentHash: %s\n", updatedItem.ContentHash[:8])
		}

		// Store the updated item back to cache
		fmt.Fprintf(os.Stderr, "  --> Calling CacheMap.Store() to save updated item...\n")
		sc.CacheMap.Store(url, updatedItem)
		fmt.Fprintf(os.Stderr, "  --> Store completed successfully!\n")

		// Verify the store worked by reading it back
		verifyValue, verifyOk := sc.CacheMap.Load(url)
		if verifyOk {
			verifyItem := verifyValue.(*CacheItem)
			fmt.Fprintf(os.Stderr, "  VERIFICATION - Read back from cache:\n")
			fmt.Fprintf(os.Stderr, "    Timestamp:   %d\n", verifyItem.Timestamp)
			fmt.Fprintf(os.Stderr, "    MaxAge:      %d\n", verifyItem.MaxAge)
			fmt.Fprintf(os.Stderr, "    BaseMaxAge:  %d\n", verifyItem.BaseMaxAge)
			fmt.Fprintf(os.Stderr, "    JSON length: %d\n", len(verifyItem.Json))
			fmt.Fprintf(os.Stderr, "    ContentHash: %s\n", verifyItem.ContentHash[:8])
		} else {
			fmt.Fprintf(os.Stderr, "  ERROR: Failed to verify - could not load item back from cache!\n")
		}

		fmt.Fprintf(os.Stderr, "=== END CACHE CHECK ===\n\n")
		return updatedItem, nil
	}

	fmt.Fprintf(os.Stderr, "  --> Cache hit - returning existing item\n")
	fmt.Fprintf(os.Stderr, "=== END CACHE CHECK ===\n\n")

	// Cache hit - return existing valid item
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
				hash := md5.Sum([]byte(rawResponse))
				contentHash := hex.EncodeToString(hash[:])

				sc.cpool.Release(conn)
				return &CacheItem{
					Timestamp:   time.Now().Unix(),
					Json:        rawResponse,
					MaxAge:      sc.MaxAge,
					BaseMaxAge:  sc.MaxAge,
					ContentHash: contentHash,
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

	hash := md5.Sum([]byte(rawResponse))
	contentHash := hex.EncodeToString(hash[:])

	sc.cpool.Release(conn)
	return &CacheItem{
		Timestamp:   time.Now().Unix(),
		Json:        jStr,
		MaxAge:      sc.MaxAge,
		BaseMaxAge:  sc.MaxAge,
		ContentHash: contentHash,
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

	err = json.Unmarshal(([]byte)(result.Json), &besapi)
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

	err = json.Unmarshal(([]byte)(result.Json), &besapi)
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

	err = json.Unmarshal(([]byte)(result.Json), &besapi)
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

// StartGarbageCollector starts a background goroutine that periodically sweeps the cache
// and frees memory by clearing Json data from expired cache items.
// The interval parameter specifies how often the garbage collector runs in seconds.
// This function should be called once after the cache is initialized with servers.
func (cache *BigFixCache) StartGarbageCollector(interval uint64) {
	if interval == 0 {
		interval = 15 // Default to 15 seconds
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	go func() {
		for range ticker.C {
			cache.sweepExpiredItems()
		}
	}()
}

// sweepExpiredItems iterates through all cache items and clears Json data from expired items
// to free memory. It replaces expired CacheItems with new ones that have empty Json but
// preserve other metadata for potential reuse.
func (cache *BigFixCache) sweepExpiredItems() {
	now := time.Now().Unix()

	cache.ServerCache.Range(func(key, value interface{}) bool {
		server := value.(*BigFixServerCache)

		server.CacheMap.Range(func(urlKey, itemValue interface{}) bool {
			item := itemValue.(*CacheItem)

			// Check if item is expired
			if now-item.Timestamp > int64(item.MaxAge) {
				// Create a new CacheItem with empty Json but preserve other fields
				clearedItem := &CacheItem{
					Timestamp:   item.Timestamp,
					Json:        "", // Clear the JSON data to free memory
					MaxAge:      item.MaxAge,
					BaseMaxAge:  item.BaseMaxAge,
					ContentHash: item.ContentHash,
				}

				// Replace the entire CacheItem for thread safety
				server.CacheMap.Store(urlKey, clearedItem)
			}

			return true
		})

		return true
	})
}
