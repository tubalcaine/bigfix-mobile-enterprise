package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

type Config struct {
	AppUser         string         `json:"app_user"`
	AppPass         string         `json:"app_pass"`
	AppCacheTimeout uint64         `json:"app_cache_timeout"`
	BigFixServers   []BigFixServer `json:"bigfix_servers"`
}

type BigFixServer struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	app_version = "0.1"
	app_desc    = "BigFix Enterprise Mobile Server"
)

func main() {
	configFile := flag.String("c", "./bem.json", "Path to the config file")
	flag.Parse()

	if *configFile == "" {
		log.Fatal("Config file not provided")
	}

	configData, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatal("Failed to read config file:", err)
	}

	var config Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		log.Fatal("Failed to parse config file:", err)
	}

	fmt.Println(app_desc)
	fmt.Println("Version " + app_version)

	cache := bfrest.GetCache(config.AppCacheTimeout)

	for _, server := range config.BigFixServers {
		go bfrest.PopulateCoreTypes(server.URL, server.Username, server.Password, 0)
	}

	// At this point we will start a web service, but for now, just loop
	// and wait for input so the program doesn't exit.
	for {
		fmt.Println("\n\nEnter a url (exit to terminate): ")
		var query string
		fmt.Scanln(&query)
		if query == "exit" {
			break
		}

		if query == "cache" {
			cache.ServerCache.Range(func(key, value interface{}) bool {
				server := value.(*bfrest.BigFixServerCache)
				fmt.Println(server.ServerName)
				server.CacheMap.Range(func(key, value interface{}) bool {
					fmt.Printf("\t%s\n", key.(string))
					return true
				})

				return true
			})
			continue
		}

		fmt.Println(cache.Get(query, config.AppUser, config.AppPass))
	}
}
