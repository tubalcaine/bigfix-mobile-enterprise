package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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
	CertPath        string         `json:"cert_path"`
	KeyPath         string         `json:"key_path"`
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
		url := c.Query("url")
		fmt.Print("URL: ", url, "\n")
		cacheItem, err := cache.Get(url)
		fmt.Print(err, "\n")

		if err == nil {
			c.JSON(200, gin.H{
				"cacheitem": cacheItem.Json,
			})
		} else {
			c.JSON(404, gin.H{
				"cacheitem": "",
				"error":     err.Error(),
			})
		}
	})

	r.GET("/servers", func(c *gin.Context) {
		var serverNames []string

		cache.ServerCache.Range(func(key, value interface{}) bool {
			server := value.(*bfrest.BigFixServerCache)
			serverNames = append(serverNames, server.ServerName)
			return true
		})

		c.JSON(200, gin.H{
			"ServerNames":     serverNames,
			"NumberOfServers": len(serverNames),
		})
	})

	r.GET("/help", func(c *gin.Context) {
		endpoints := []string{
			"/urls",
			"/servers",
			"/summary",
			"/cache",
			"/help",
		}
		htmlContent := "<html><body><h1>Available Endpoints</h1><ul>"
		for _, endpoint := range endpoints {
			htmlContent += "<li>" + endpoint + "</li>"
		}
		htmlContent += "</ul></body></html>"
		c.Data(200, "text/html; charset=utf-8", []byte(htmlContent))
	})

	r.GET("/summary", func(c *gin.Context) {
		summary := make(map[string]interface{})
		var totalSize int64

		cache.ServerCache.Range(func(key, value interface{}) bool {
			server := value.(*bfrest.BigFixServerCache)
			serverSummary := make(map[string]interface{})
			count, current, expired := 0, 0, 0
			var serverSize int64

			server.CacheMap.Range(func(key, value interface{}) bool {
				v := value.(*bfrest.CacheItem)
				count++
				itemSize := int64(len(v.Json) + len(v.RawXML))
				serverSize += itemSize
				if time.Now().Unix()-v.Timestamp > int64(server.MaxAge) {
					expired++
				} else {
					current++
				}
				return true
			})

			serverSummary["total_items"] = count
			serverSummary["expired_items"] = expired
			serverSummary["current_items"] = current
			serverSummary["serverSize"] = serverSize
			summary[server.ServerName] = serverSummary
			totalSize += serverSize

			return true
		})

		summary["totalSize"] = totalSize
		c.JSON(200, summary)
	})

	r.GET("/cache", func(c *gin.Context) {
		cacheData := make(map[string][]string)

		cache.ServerCache.Range(func(key, value interface{}) bool {
			server := value.(*bfrest.BigFixServerCache)
			var cacheItems []string

			server.CacheMap.Range(func(key, value interface{}) bool {
				cacheItems = append(cacheItems, key.(string))
				return true
			})

			cacheData[server.ServerName] = cacheItems
			return true
		})

		c.JSON(200, cacheData)
	})

	// Configure TLS
	// Remove the declaration of tlsConfig since it is not being used
	// tlsConfig := &tls.Config{
	//     MinVersion: tls.VersionTLS12,
	// }

	if config.KeyPath != "" {
		go r.RunTLS(":"+strconv.Itoa(config.ListenPort), config.CertPath, config.KeyPath) // listen and serve on specified port with TLS
	} else {
		go r.Run(":" + strconv.Itoa(config.ListenPort)) // listen and serve on specified port
	}

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

		if query == "makekey" {
			fmt.Println("Enter the key file name:")
			var keyFileName string
			fmt.Scanln(&keyFileName)

			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				fmt.Println("Error generating private key:", err)
				continue
			}

			publicKey := &privateKey.PublicKey

			// Save private key to file
			privateKeyFile, err := os.Create(keyFileName + ".key")
			if err != nil {
				fmt.Println("Error creating private key file:", err)
				continue
			}
			defer privateKeyFile.Close()

			privateKeyPEM := &pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
			}
			err = pem.Encode(privateKeyFile, privateKeyPEM)
			if err != nil {
				fmt.Println("Error encoding private key:", err)
				continue
			}

			// Save public key to file
			publicKeyFile, err := os.Create(keyFileName + ".pub")
			if err != nil {
				fmt.Println("Error creating public key file:", err)
				continue
			}
			defer publicKeyFile.Close()

			publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
			if err != nil {
				fmt.Println("Error marshaling public key:", err)
				continue
			}

			publicKeyPEM := &pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: publicKeyBytes,
			}
			err = pem.Encode(publicKeyFile, publicKeyPEM)
			if err != nil {
				fmt.Println("Error encoding public key:", err)
				continue
			}

			fmt.Println("Key pair generated successfully.")
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
					itemData["Json"] = cacheItem.Json
					itemData["RawXML"] = cacheItem.RawXML
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
			fmt.Println("\tmakekey - generate a new RSA key pair for client authentication")
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
