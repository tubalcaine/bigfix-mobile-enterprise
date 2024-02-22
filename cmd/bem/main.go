package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

type Config struct {
	AppCacheTimeout uint64         `json:"app_cache_timeout"`
	BigFixServers   []BigFixServer `json:"bigfix_servers"`
	ListenPort      int            `json:"listen_port"`
}

type BigFixServer struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	MaxAge   uint64 `json:"maxage"`
	PoolSize int    `json:"poolsize"`
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

	cache := bfrest.
		GetCache(config.AppCacheTimeout)

	for _, server := range config.BigFixServers {
		cache.AddServer(server.URL, server.Username, server.Password, server.PoolSize)
		go cache.PopulateCoreTypes(server.URL, server.MaxAge)
	}

	r := gin.Default()

	r.GET("/urls", func(c *gin.Context) {
		// TODO: Implement the handler
		c.JSON(200, gin.H{
			"message": "urls",
		})
	})

	r.GET("/servers", func(c *gin.Context) {
		// TODO: Implement the handler
		c.JSON(200, gin.H{
			"message": "servers",
		})
	})

	r.GET("/help", func(c *gin.Context) {
		// TODO: Implement the handler
		c.JSON(200, gin.H{
			"message": "help",
		})
	})

	r.GET("/summary", func(c *gin.Context) {
		// TODO: Implement the handler
		c.JSON(200, gin.H{
			"message": "summary",
		})
	})

	r.GET("/cache", func(c *gin.Context) {
		// TODO: Implement the handler
		c.JSON(200, gin.H{
			"message": "cache",
		})
	})

	// Run the web server in a goroutine so we can continue to process
	// input from the user.

	go r.Run(":" + strconv.Itoa(config.ListenPort)) // listen and serve on specified port

	// loop and wait for input so the program doesn't exit.
	for {
		fmt.Println("\n\nEnter a url or command (exit to terminate): ")
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

		if query == "write" {
			fmt.Println("Enter the file name:")
			var fileName string
			fmt.Scanln(&fileName)

			file, err := os.Create(fileName)
			if err != nil {
				fmt.Println("Error creating file:", err)
				continue
			}
			defer file.Close()

			cache.ServerCache.Range(func(key, value interface{}) bool {
				server := value.(*bfrest.BigFixServerCache)
				serverData := make(map[string]interface{})
				serverData["ServerName"] = server.ServerName
				serverData["CacheItems"] = make([]map[string]interface{}, 0)

				server.CacheMap.Range(func(key, value interface{}) bool {
					cacheItem := value.(*bfrest.CacheItem)
					itemData := make(map[string]interface{})
					itemData["Key"] = key.(string)
					itemData["Value"] = cacheItem.Json
					itemData["Timestamp"] = cacheItem.Timestamp

					serverData["CacheItems"] = append(serverData["CacheItems"].([]map[string]interface{}), itemData)

					return true
				})

				jsonData, err := json.MarshalIndent(serverData, "", "\t")
				if err != nil {
					fmt.Println("Error marshaling JSON:", err)
					return true
				}

				_, err = file.Write(jsonData)
				if err != nil {
					fmt.Println("Error writing to file:", err)
					return true
				}

				return true
			})

			fmt.Println("Cache written to file:", fileName)
			continue
		}

		if query == "summary" {
			cache.ServerCache.Range(func(key, value interface{}) bool {
				server := value.(*bfrest.BigFixServerCache)
				fmt.Printf("For server %s\n\tWe have:\n", server.ServerName)
				count, current, expired := 0, 0, 0
				server.CacheMap.Range(func(key, value interface{}) bool {
					v := value.(*bfrest.CacheItem)
					count++
					if time.Now().Unix()-v.Timestamp > int64(server.MaxAge) {
						expired++
					} else {
						current++
					}

					return true
				})

				fmt.Printf("\t\t%d total items, %d expired, %d current\n\n", count, expired, current)

				return true
			})
			continue
		}

		if query == "help" {
			fmt.Println("Commands:")
			fmt.Println("\tcache - display the current cache")
			fmt.Println("\tsummary - display a summary of the cache")
			fmt.Println("\twrite - write the cache to a file")
			fmt.Println("\thelp - display this help")
			fmt.Println("\texit - terminate the program")
			fmt.Println("\t<url> - retrieve the url from the cache")
			continue
		}

		if !strings.HasPrefix(query, "http") {
			continue
		}

		fmt.Println(cache.Get(query))
	}
}
