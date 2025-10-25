package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/gin-gonic/gin"
	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

func main() {
	configFile := flag.String("c", "./bem.json", "Path to the config file")
	showVersion := flag.Bool("version", false, "Display version information and exit")
	flag.Parse()

	// Handle --version flag
	if *showVersion {
		fmt.Println(VersionString())
		os.Exit(0)
	}

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
		"version", ShortVersion(),
		"build_date", BuildDate,
		"commit", GitCommit)

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

	// Set cache debug output based on log level
	// Cache debug is enabled only when log level is DEBUG
	if config.LogLevel != "" {
		// Use log_level field if set
		if strings.ToUpper(config.LogLevel) == "DEBUG" {
			cache.Debug = 1
		} else {
			cache.Debug = 0
		}
	} else {
		// Backward compatibility: fallback to debug field if log_level not set
		cache.Debug = config.Debug
	}

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

	// Keep Gin in debug mode (default) to enable colorized [GIN] HTTP request logs
	// These logs always go to console regardless of log_level setting
	// Note: This is separate from application log level which controls slog output

	// Create Gin router with custom middleware
	r := gin.New()

	// Configure Gin to always write colorized HTTP request logs to console
	gin.DefaultWriter = GetGinLogWriter()       // Always os.Stdout
	gin.DefaultErrorWriter = GetGinLogWriter()  // Always os.Stdout

	// Add Gin's default logger middleware (provides colorized [GIN] HTTP request logs to console)
	r.Use(gin.LoggerWithWriter(GetGinLogWriter()))

	// Add structured slog request logging middleware (logs all requests at INFO level via slog)
	// This goes to file/console based on log_to_file/log_to_console settings
	logger := GetLogger()
	r.Use(RequestLoggingMiddleware(logger))

	// Add recovery and error logging middleware
	r.Use(RecoveryMiddleware(logger))
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

	// Initialize readline for command history and line editing
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "bem> ",
		HistoryFile:     "/tmp/.bem_history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		slog.Error("Failed to initialize readline", "error", err)
		os.Exit(1)
	}
	defer rl.Close()

	slog.Info("Interactive CLI started",
		"prompt", "bem>",
		"history_file", "/tmp/.bem_history",
		"tip", "Use up/down arrows for command history, 'exit' to quit")

	// loop and wait for input so the program doesn't exit.
	for {
		fmt.Println() // Print newline before prompt
		query, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			break
		}

		query = strings.TrimSpace(query)

		if query == "" {
			continue // Skip empty lines
		}

		if query == "exit" {
			break
		}

		if query == "cache" {
			const itemsPerPage = 10

			cache.ServerCache.Range(func(key, value interface{}) bool {
				server := value.(*bfrest.BigFixServerCache)
				fmt.Printf("\n=== Server: %s ===\n", server.ServerName)

				// Collect all cache items first for pagination
				type cacheEntry struct {
					url  string
					item *bfrest.CacheItem
				}
				var entries []cacheEntry

				server.CacheMap.Range(func(key, value interface{}) bool {
					entries = append(entries, cacheEntry{
						url:  key.(string),
						item: value.(*bfrest.CacheItem),
					})
					return true
				})

				totalItems := len(entries)
				if totalItems == 0 {
					fmt.Println("  (no cache items)")
					return true
				}

				// Display items with pagination
				for i, entry := range entries {
					// Calculate remaining time
					now := time.Now().Unix()
					age := now - entry.item.Timestamp
					remaining := int64(entry.item.MaxAge) - age
					if remaining < 0 {
						remaining = 0
					}

					// Truncate hash for display
					hashDisplay := entry.item.ContentHash
					if len(hashDisplay) > 12 {
						hashDisplay = hashDisplay[:12] + "..."
					}

					fmt.Printf("\n  URL: %s\n", entry.url)
					fmt.Printf("    MaxAge: %d seconds\n", entry.item.MaxAge)
					fmt.Printf("    Content Hash: %s\n", hashDisplay)
					fmt.Printf("    Remaining Time: %d seconds %s\n", remaining, func() string {
						if remaining == 0 {
							return "(EXPIRED)"
						}
						return ""
					}())
					fmt.Printf("    Hit Count: %d\n", entry.item.HitCount)
					fmt.Printf("    Miss Count: %d\n", entry.item.MissCount)

					// Check if we should pause for pagination
					itemNum := i + 1
					if itemNum%itemsPerPage == 0 && itemNum < totalItems {
						fmt.Printf("\n--- Showing %d of %d items. Press ENTER for more, or 'c' then ENTER to continue: ", itemNum, totalItems)
						rl.SetPrompt("")
						input, err := rl.Readline()
						rl.SetPrompt("bem> ") // Reset prompt
						if err != nil {
							break
						}
						if strings.ToLower(strings.TrimSpace(input)) == "c" {
							fmt.Println("(continuing without pagination...)")
							// Set itemsPerPage to a very large number to skip future pauses
							// We can't modify the const, so we'll just break and print remaining
							for j := i + 1; j < len(entries); j++ {
								entry := entries[j]
								now := time.Now().Unix()
								age := now - entry.item.Timestamp
								remaining := int64(entry.item.MaxAge) - age
								if remaining < 0 {
									remaining = 0
								}

								hashDisplay := entry.item.ContentHash
								if len(hashDisplay) > 12 {
									hashDisplay = hashDisplay[:12] + "..."
								}

								fmt.Printf("\n  URL: %s\n", entry.url)
								fmt.Printf("    MaxAge: %d seconds\n", entry.item.MaxAge)
								fmt.Printf("    Content Hash: %s\n", hashDisplay)
								fmt.Printf("    Remaining Time: %d seconds %s\n", remaining, func() string {
									if remaining == 0 {
										return "(EXPIRED)"
									}
									return ""
								}())
								fmt.Printf("    Hit Count: %d\n", entry.item.HitCount)
								fmt.Printf("    Miss Count: %d\n", entry.item.MissCount)
							}
							break
						}
					}
				}
				fmt.Println()

				return true
			})
			continue
		}

		if query == "write" {
			fmt.Print("Enter the file name: ")
			rl.SetPrompt("")
			fileName, err := rl.Readline()
			rl.SetPrompt("bem> ") // Reset prompt
			if err != nil {
				fmt.Println("Error reading filename")
				continue
			}
			fileName = strings.TrimSpace(fileName)
			if fileName == "" {
				fmt.Println("Filename cannot be empty")
				continue
			}

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
				var totalHits, totalMisses uint64
				var maxAge, minAge uint64
				firstItem := true

				server.CacheMap.Range(func(key, value interface{}) bool {
					v := value.(*bfrest.CacheItem)
					count++
					ramBytes += int64(len(v.Json))

					// Track hits and misses
					totalHits += v.HitCount
					totalMisses += v.MissCount

					// Track MaxAge min/max
					if firstItem {
						maxAge = v.MaxAge
						minAge = v.MaxAge
						firstItem = false
					} else {
						if v.MaxAge > maxAge {
							maxAge = v.MaxAge
						}
						if v.MaxAge < minAge {
							minAge = v.MaxAge
						}
					}

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
				fmt.Printf("\t\tRAM usage: %.2f KB (%.2f MB)\n", ramKB, ramMB)
				fmt.Printf("\t\tMaxAge range: %d to %d seconds\n", minAge, maxAge)
				fmt.Printf("\t\tCache hits: %d, Cache misses: %d\n\n", totalHits, totalMisses)

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