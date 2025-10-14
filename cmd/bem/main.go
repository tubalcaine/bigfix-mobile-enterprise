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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
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

	// Set up configuration directory for persistent storage
	configDir = filepath.Dir(*configFile)
	
	// Set default registration data directory if not configured
	registrationDataDir = config.RegistrationDataDir
	if registrationDataDir == "" {
		registrationDataDir = configDir // fallback to config directory
	}
	
	// Load existing registration data
	if err := loadRegistrationOTPs(); err != nil {
		log.Fatal("Failed to load registration OTPs:", err)
	}
	
	if err := loadRegisteredClients(); err != nil {
		log.Fatal("Failed to load registered clients:", err)
	}
	
	fmt.Printf("Loaded %d registration OTPs and %d registered clients\n", len(registrationOTPs), len(registeredClients))
	
	// Start registration directory monitoring
	go watchRegistrationDirectory(config.RegistrationDir)
	
	// Start periodic session cleanup (every 30 minutes)
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredSessions()
		}
	}()

	cache := bfrest.GetCache(config.AppCacheTimeout, config.MaxCacheLifetime)

	for _, server := range config.BigFixServers {
		cache.AddServer(server.URL, server.Username, server.Password, server.PoolSize)
		go cache.PopulateCoreTypes(server.URL, server.MaxAge)
	}

	// Start the garbage collector after cache is initialized with servers
	cache.StartGarbageCollector(config.GarbageCollectorInterval)

	r := gin.Default()

	// Set up all routes
	setupRoutes(r, cache, config)

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

			// Handle the case where key size is not provided in the config file
			keySize := config.KeySize
			if keySize == 0 {
				keySize = 2048
			}
			privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
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

		if query == "registrations" {
			fmt.Println("\n=== REGISTRATION STATUS ===")
			
			// Registration Requests (OTPs)
			registrationMutex.RLock()
			fmt.Printf("\nRegistration Requests (%d):\n", len(registrationOTPs))
			if len(registrationOTPs) == 0 {
				fmt.Println("  (none)")
			} else {
				for i, otp := range registrationOTPs {
					fmt.Printf("  %d. %s\n", i+1, otp.ClientName)
					fmt.Printf("     Key: %s\n", otp.OneTimeKey)
					fmt.Printf("     Created: %s\n", otp.CreatedAt.Format("2006-01-02 15:04:05"))
					fmt.Printf("     Lifespan: %d days\n", otp.KeyLifespanDays)
					if otp.RequestedBy != "" {
						fmt.Printf("     Requested by: %s\n", otp.RequestedBy)
					}
					fmt.Println()
				}
			}
			
			// Registered Clients  
			fmt.Printf("Registered Clients (%d):\n", len(registeredClients))
			if len(registeredClients) == 0 {
				fmt.Println("  (none)")
			} else {
				for i, client := range registeredClients {
					fmt.Printf("  %d. %s\n", i+1, client.ClientName)
					fmt.Printf("     Registered: %s\n", client.RegisteredAt.Format("2006-01-02 15:04:05"))
					if client.ExpiresAt != nil {
						fmt.Printf("     Expires: %s\n", client.ExpiresAt.Format("2006-01-02 15:04:05"))
					} else {
						fmt.Printf("     Expires: Never\n")
					}
					fmt.Printf("     Last Used: %s\n", client.LastUsed.Format("2006-01-02 15:04:05"))
					fmt.Printf("     Key Lifespan: %d days\n", client.KeyLifespanDays)
					fmt.Println()
				}
			}
			registrationMutex.RUnlock()
			
			// Active Sessions
			sessionMutex.RLock()
			fmt.Printf("Active OTP Sessions (%d):\n", len(activeSessions))
			if len(activeSessions) == 0 {
				fmt.Println("  (none)")
			} else {
				i := 1
				now := time.Now()
				for token, expiresAt := range activeSessions {
					status := "Active"
					if now.After(expiresAt) {
						status = "Expired"
					}
					fmt.Printf("  %d. Session Token: %s...\n", i, token[:8])
					fmt.Printf("     Expires: %s\n", expiresAt.Format("2006-01-02 15:04:05"))
					fmt.Printf("     Status: %s\n", status)
					fmt.Println()
					i++
				}
			}
			sessionMutex.RUnlock()
			
			continue
		}

		if query == "help" {
			fmt.Println("Commands:")
			fmt.Println("\tcache - display the current cache")
			fmt.Println("\tsummary - display a summary of the cache")
			fmt.Println("\twrite - write the cache to a file")
			fmt.Println("\tmakekey - generate a new RSA key pair for client authentication")
			fmt.Println("\tregistrations - display registration requests, clients, and sessions")
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