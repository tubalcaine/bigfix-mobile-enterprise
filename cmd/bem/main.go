package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

func main() {
	configFile := flag.String("c", "./bem.json", "Path to the config file")
	flag.Parse()

	if *configFile == "" {
		slog.Error("Config file not provided")
		os.Exit(1)
	}

	configData, err := os.ReadFile(*configFile)
	if err != nil {
		slog.Error("Failed to read config file", "error", err)
		os.Exit(1)
	}

	var config Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		slog.Error("Failed to parse config file", "error", err)
		os.Exit(1)
	}

	// Make config globally accessible
	appConfig = &config

	// Initialize logger with full config
	if err := InitLogger(config); err != nil {
		slog.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting application",
		"name", app_desc,
		"version", app_version)

	// Set up configuration directory for persistent storage
	configDir = filepath.Dir(*configFile)
	
	// Set default registration data directory if not configured
	registrationDataDir = config.RegistrationDataDir
	if registrationDataDir == "" {
		registrationDataDir = configDir // fallback to config directory
	}

	// Load existing registration data
	if err := loadRegistrationOTPs(); err != nil {
		slog.Error("Failed to load registration OTPs", "error", err)
		os.Exit(1)
	}

	if err := loadRegisteredClients(); err != nil {
		slog.Error("Failed to load registered clients", "error", err)
		os.Exit(1)
	}

	slog.Debug("Loaded registration data",
		"otp_count", len(registrationOTPs),
		"client_count", len(registeredClients))
	
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
	cache.Debug = config.Debug

	slog.Info("Initializing BigFix server connections",
		"server_count", len(config.BigFixServers))

	for _, server := range config.BigFixServers {
		cache.AddServer(server.URL, server.Username, server.Password, server.PoolSize, server.MaxAge)
		go cache.PopulateCoreTypes(server.URL, server.MaxAge)
		slog.Debug("Added BigFix server",
			"url", server.URL,
			"pool_size", server.PoolSize,
			"max_age", server.MaxAge)
	}

	// Start the garbage collector after cache is initialized with servers
	cache.StartGarbageCollector(config.GarbageCollectorInterval)

	// Set Gin to release mode if not in debug
	if config.Debug == 0 {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router with custom middleware
	r := gin.New()

	// Add our custom middleware
	logger := GetLogger()
	r.Use(RecoveryMiddleware(logger))
	r.Use(RequestLoggingMiddleware(logger))
	r.Use(ErrorLoggingMiddleware(logger))

	// Set up all routes
	setupRoutes(r, cache, config)

	// Validate TLS configuration (HTTPS-only server)
	if config.KeyPath == "" || config.CertPath == "" {
		slog.Error("TLS certificate and key are required - HTTP-only mode is not supported")
		os.Exit(1)
	}

	// Start HTTPS server in goroutine
	go func() {
		err := StartTLSServer(r, config.CertPath, config.KeyPath, config.ListenPort, logger)
		if err != nil {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

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
				serverData["MaxAge"] = server.MaxAge
				serverData["CacheItems"] = make([]map[string]interface{}, 0)

				server.CacheMap.Range(func(key, value interface{}) bool {
					cacheItem := value.(*bfrest.CacheItem)
					itemData := make(map[string]interface{})
					itemData["URL"] = key.(string)
					itemData["Timestamp"] = cacheItem.Timestamp
					itemData["MaxAge"] = cacheItem.MaxAge
					itemData["BaseMaxAge"] = cacheItem.BaseMaxAge
					itemData["ContentHash"] = cacheItem.ContentHash
					itemData["Json"] = cacheItem.Json

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
				var ramBytes int64
				server.CacheMap.Range(func(key, value interface{}) bool {
					v := value.(*bfrest.CacheItem)
					count++
					ramBytes += int64(len(v.Json))
					if time.Now().Unix()-v.Timestamp > int64(server.MaxAge) {
						expired++
					} else {
						current++
					}

					return true
				})

				ramKB := float64(ramBytes) / 1024.0
				ramMB := ramKB / 1024.0
				fmt.Printf("\t\t%d total items, %d expired, %d current\n", count, expired, current)
				fmt.Printf("\t\tRAM usage: %.2f KB (%.2f MB)\n\n", ramKB, ramMB)

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

		if query == "reload" {
			fmt.Println("Reloading cache with core types from all servers...")
			for _, server := range config.BigFixServers {
				go cache.PopulateCoreTypes(server.URL, server.MaxAge)
				fmt.Printf("  Started cache population for: %s\n", server.URL)
			}
			fmt.Println("Cache reload initiated for all servers.")
			continue
		}

		if query == "help" {
			fmt.Println("Commands:")
			fmt.Println("\tcache - display the current cache")
			fmt.Println("\tsummary - display a summary of the cache")
			fmt.Println("\twrite - write the cache to a file")
			fmt.Println("\tmakekey - generate a new RSA key pair for client authentication")
			fmt.Println("\tregistrations - display registration requests, clients, and sessions")
			fmt.Println("\treload - re-populate cache with core types from all servers")
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