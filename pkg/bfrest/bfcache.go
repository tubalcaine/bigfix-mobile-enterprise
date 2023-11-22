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
	sc     sync.Map
	maxAge uint64
}

type BigFixServerCache struct {
	serverName string
	cpool      *Pool
	cacheMap   sync.Map
	maxAge     uint64
}

type CacheItem struct {
	Timestamp int64
	RawXML    string
	Json      string
}

var cacheInstance *BigFixCache

// Singleton cache constructor
func GetCache() *BigFixCache {
	if cacheInstance == nil {
		cacheInstance = &BigFixCache{
			sc:     sync.Map{},
			maxAge: 300,
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

func silentGet(url, username, passwd string) {
	fmt.Fprintf(os.Stderr, "Silent GET URL: %s\n", url)
	res, err := Get(url, username, passwd)

	if err != nil {
		fmt.Fprintf(os.Stderr, "For URL: %s\n", url)
		fmt.Fprintln(os.Stderr, res)
		fmt.Fprintf(os.Stderr, "Silent GET failed: %s\n", err)
		os.Exit(1)
	}
}

func Get(url, username, passwd string) (*CacheItem, error) {
	baseURL := getBaseUrl(url)

	fmt.Fprintf(os.Stderr, "Get URL: %s\n", url)

	cache := GetCache()

	// cacheMutex.Lock()
	scValue, err := cache.sc.Load(baseURL)
	if !err {
		newpool, _ := NewPool(baseURL, username, passwd, 8)

		scInstance := &BigFixServerCache{
			serverName: baseURL,
			cpool:      newpool,
			maxAge:     cache.maxAge,
			cacheMap:   sync.Map{},
		}

		cache.sc.Store(baseURL, scInstance)
		scValue, _ = cache.sc.Load(baseURL)
	}

	sc, _ := scValue.(*BigFixServerCache)

	// If the result doesn't exist or is too old, pull it from the server
	value, err := sc.cacheMap.Load(url)

	var cm *CacheItem

	// Cache miss
	if !err {
		cm, err := retrieveBigFixData(url, sc)
		if err != nil {
			return nil, err
		}
		sc.cacheMap.Store(url, cm)
		return cm, nil
	}

	cm, err = value.(*CacheItem)

	if err {
		return nil, fmt.Errorf("type failure loading cache item for %s", url)
	}

	// Cache expired
	if time.Now().Unix()-cm.Timestamp > int64(sc.maxAge) {
		cm, err := retrieveBigFixData(url, sc)
		if err != nil {
			return nil, err
		}
		sc.cacheMap.Store(url, cm)
		return cm, nil
	}

	// Cache hit
	return cm, nil
}

func retrieveBigFixData(url string, sc *BigFixServerCache) (*CacheItem, error) {
	conn, err := sc.cpool.Acquire()

	if err != nil {
		return nil, err
	}

	defer sc.cpool.Release(conn)

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
		}

		jsonValue, err = json.Marshal(&bes)
		if err != nil {
			return nil, err
		}
	}

	jStr := (string)(jsonValue)

	return &CacheItem{
		Timestamp: time.Now().Unix(),
		RawXML:    rawXML,
		Json:      jStr,
	}, nil
}

func PopulateCoreTypes(serverUrl string, username string, password string) error {
	var besapi BESAPI
	result, err := Get(serverUrl+"/api/actions", username, password)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, action := range besapi.Action {
		//		silentGet(action.Resource, username, password)
		go silentGet(action.Resource, username, password)
	}

	result, err = Get(serverUrl+"/api/computers", username, password)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, computer := range besapi.Computer {
		//		silentGet(computer.Resource, username, password)
		go silentGet(computer.Resource, username, password)
	}

	result, err = Get(serverUrl+"/api/sites", username, password)

	if err != nil {
		return err
	}

	err = xml.Unmarshal(([]byte)(result.RawXML), &besapi)
	if err != nil {
		return err
	}

	for _, site := range besapi.CustomSite {
		//		silentGet(site.Resource, username, password)
		go silentGet(site.Resource, username, password)
	}

	return err
}
