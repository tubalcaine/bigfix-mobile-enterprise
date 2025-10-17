package main

import (
	"encoding/json"
	"fmt"
	"log"
	neturl "net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

// HTTP endpoint handlers

func setupRoutes(r *gin.Engine, cache *bfrest.BigFixCache, config Config) {
	// OTP endpoint for admin session creation (no authentication required)
	r.GET("/otp", handleOTPEndpoint)
	r.POST("/otp", handleOTPEndpoint)

	// Client registration endpoint (no authentication required)
	r.POST("/register", func(c *gin.Context) {
		handleRegisterEndpoint(c, config)
	})
	r.GET("/register", func(c *gin.Context) {
		handleRegisterEndpoint(c, config)
	})

	// Registration request endpoint (no authentication required)
	r.GET("/requestregistration", func(c *gin.Context) {
		handleRegistrationRequest(c, config)
	})
	r.POST("/requestregistration", func(c *gin.Context) {
		handleRegistrationRequest(c, config)
	})

	// Help endpoint (no authentication required)
	r.GET("/help", handleHelpEndpoint)
	r.POST("/help", handleHelpEndpoint)
	
	// Debug endpoint - temporary, no auth required
	r.GET("/debug/servers", func(c *gin.Context) {
		handleDebugServersEndpoint(c, cache)
	})
	r.POST("/debug/servers", func(c *gin.Context) {
		handleDebugServersEndpoint(c, cache)
	})

	// Protected endpoints require authentication
	r.GET("/urls", func(c *gin.Context) {
		handleURLsEndpoint(c, cache)
	})
	r.POST("/urls", func(c *gin.Context) {
		handleURLsEndpoint(c, cache)
	})

	r.GET("/servers", func(c *gin.Context) {
		handleServersEndpoint(c, cache)
	})
	r.POST("/servers", func(c *gin.Context) {
		handleServersEndpoint(c, cache)
	})

	r.GET("/summary", func(c *gin.Context) {
		handleSummaryEndpoint(c, cache)
	})
	r.POST("/summary", func(c *gin.Context) {
		handleSummaryEndpoint(c, cache)
	})

	r.GET("/cache", func(c *gin.Context) {
		handleCacheEndpoint(c, cache)
	})
	r.POST("/cache", func(c *gin.Context) {
		handleCacheEndpoint(c, cache)
	})
}

func handleOTPEndpoint(c *gin.Context) {
	oneTimeKey := c.Query("OneTimeKey")
	if oneTimeKey == "" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "OneTimeKey parameter is required",
		})
		return
	}

	// Find and remove OTP (consumes it like registration does)
	otp, found := findAndRemoveOTPByKey(oneTimeKey)
	if !found {
		log.Printf("Failed admin session attempt with invalid OTP: %s", oneTimeKey)
		c.JSON(401, gin.H{
			"success": false,
			"message": "Invalid OneTimeKey",
		})
		return
	}

	// Create admin session and set cookie
	sessionToken := createAdminSession(*otp)
	
	c.SetCookie("bem_session", sessionToken, 8*60*60, "/", "", false, true) // 8 hours, HttpOnly
	
	// Save updated OTPs (with the used one removed)
	if err := saveRegistrationOTPs(); err != nil {
		log.Printf("Error saving registration OTPs after admin session creation: %v", err)
	}

	log.Printf("Admin session created using OTP for: %s", otp.ClientName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "Admin session created successfully",
		"expires": "8 hours from now",
	})
}

func handleRegisterEndpoint(c *gin.Context, config Config) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, RegisterResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.ClientName == "" || req.OneTimeKey == "" {
		c.JSON(400, RegisterResponse{
			Success: false,
			Message: "ClientName and OneTimeKey are required",
		})
		return
	}

	// Check if client is already registered
	if isClientRegistered(req.ClientName) {
		c.JSON(409, RegisterResponse{
			Success: false,
			Message: "Client already registered",
		})
		return
	}

	// Find and validate OTP
	otp, found := findAndRemoveOTP(req.ClientName, req.OneTimeKey)
	if !found {
		log.Printf("Failed registration attempt from %s with invalid OTP", req.ClientName)
		c.JSON(401, RegisterResponse{
			Success: false,
			Message: "Invalid ClientName or OneTimeKey",
		})
		c.Abort() // Terminate connection as specified
		return
	}

	// Generate key pair and register client
	response, err := generateAndRegisterClient(*otp, config.KeySize)
	if err != nil {
		log.Printf("Failed to register client %s: %v", req.ClientName, err)
		c.JSON(500, RegisterResponse{
			Success: false,
			Message: "Internal server error during registration",
		})
		return
	}

	log.Printf("Successfully registered client: %s", req.ClientName)
	c.JSON(200, response)
}

func handleHelpEndpoint(c *gin.Context) {
	endpoints := []string{
		"/otp",
		"/requestregistration",
		"/register",
		"/help",
		"--- Protected endpoints (require authentication) ---",
		"/urls",
		"/servers", 
		"/summary",
		"/cache",
	}
	htmlContent := "<html><body><h1>Available Endpoints</h1><ul>"
	for _, endpoint := range endpoints {
		htmlContent += "<li>" + endpoint + "</li>"
	}
	htmlContent += "</ul></body></html>"
	c.Data(200, "text/html; charset=utf-8", []byte(htmlContent))
}

func handleURLsEndpoint(c *gin.Context, cache *bfrest.BigFixCache) {
	if !requireAuth(c) {
		return
	}
	
	var url string
	
	// Handle both GET and POST methods
	if c.Request.Method == "POST" {
		// For POST requests, expect JSON body with url field
		var requestBody struct {
			URL string `json:"url"`
		}
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(400, gin.H{
				"error": "Invalid JSON body. Expected {\"url\": \"...\"}",
			})
			return
		}
		url = requestBody.URL
		if appConfig.Debug != 0 {
			fmt.Printf("POST /urls - URL from body: %s\n", url)
		}
	} else {
		// For GET requests, get URL from query parameter (existing behavior)
		url = c.Query("url")
		if appConfig.Debug != 0 {
			fmt.Printf("GET /urls - URL from query: %s\n", url)
		}
	}
	
	if url == "" {
		c.JSON(400, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	// Determine if this will be a cache hit before calling cache.Get()
	requestTime := time.Now().Unix()
	isCacheHit := false

	// Parse base URL to check cache status
	if parsedURL, parseErr := neturl.Parse(url); parseErr == nil {
		baseURL := parsedURL.Scheme + "://" + parsedURL.Host
		if scValue, ok := cache.ServerCache.Load(baseURL); ok {
			sc := scValue.(*bfrest.BigFixServerCache)
			if value, ok := sc.CacheMap.Load(url); ok {
				if cm, ok := value.(*bfrest.CacheItem); ok {
					isEmpty := cm.Json == ""
					isExpired := requestTime-cm.Timestamp > int64(cm.MaxAge)
					isCacheHit = !isEmpty && !isExpired
				}
			}
		}
	}

	if appConfig.Debug != 0 {
		fmt.Printf("Processing cache request for URL: %s (will be cache hit: %v)\n", url, isCacheHit)
	}
	cacheItem, err := cache.Get(url)

	if appConfig.Debug != 0 {
		if err != nil {
			fmt.Printf("Cache error: %v\n", err)
		} else {
			fmt.Printf("Cache hit successful\n")
		}
	}

	if err == nil {
		// Check if this is a JSON passthrough (from output=json requests)
		// by looking at the URL to see if it contains output=json
		var responseData interface{}
		if strings.Contains(url, "output=json") || strings.Contains(url, "format=json") {
			// For JSON format requests, the cacheItem.Json contains raw JSON from BigFix
			// We need to parse it and include it directly to avoid double-encoding
			var jsonData interface{}
			if jsonErr := json.Unmarshal([]byte(cacheItem.Json), &jsonData); jsonErr == nil {
				responseData = jsonData
			} else {
				// If parsing fails, fall back to string
				responseData = cacheItem.Json
			}
		} else {
			// For XML->JSON conversion, return as string (existing behavior)
			responseData = cacheItem.Json
		}

		// Calculate TTL (time-to-live): timestamp + maxage - current_time
		currentTime := time.Now().Unix()
		ttl := cacheItem.Timestamp + int64(cacheItem.MaxAge) - currentTime
		if ttl < 0 {
			ttl = 0 // TTL cannot be negative
		}

		c.JSON(200, gin.H{
			"cacheitem":   responseData,
			"iscachehit":  isCacheHit,
			"timestamp":   cacheItem.Timestamp,
			"maxage":      cacheItem.MaxAge,
			"ttl":         ttl,
			"hitcount":    cacheItem.HitCount,
			"misscount":   cacheItem.MissCount,
			"contenthash": cacheItem.ContentHash,
		})
	} else {
		c.JSON(404, gin.H{
			"cacheitem": "",
			"error":     err.Error(),
		})
	}
}

// Debug endpoint to check server cache without authentication
func handleDebugServersEndpoint(c *gin.Context, cache *bfrest.BigFixCache) {
	var serverNames []string
	
	cache.ServerCache.Range(func(key, value interface{}) bool {
		server := value.(*bfrest.BigFixServerCache)
		serverNames = append(serverNames, server.ServerName)
		return true
	})
	
	c.JSON(200, gin.H{
		"debug": "no-auth-required",
		"ServerNames":     serverNames,
		"NumberOfServers": len(serverNames),
		"message": "This is a debug endpoint. Remove in production.",
	})
}

func handleServersEndpoint(c *gin.Context, cache *bfrest.BigFixCache) {
	if !requireAuth(c) {
		return
	}

	type ServerInfo struct {
		Name   string  `json:"name"`
		RAMBytes int64   `json:"ram_bytes"`
		RAMKB    float64 `json:"ram_kb"`
		RAMMB    float64 `json:"ram_mb"`
	}

	var servers []ServerInfo

	cache.ServerCache.Range(func(key, value interface{}) bool {
		server := value.(*bfrest.BigFixServerCache)
		var ramBytes int64

		server.CacheMap.Range(func(key, value interface{}) bool {
			item := value.(*bfrest.CacheItem)
			ramBytes += int64(len(item.Json))
			return true
		})

		servers = append(servers, ServerInfo{
			Name:     server.ServerName,
			RAMBytes: ramBytes,
			RAMKB:    float64(ramBytes) / 1024.0,
			RAMMB:    float64(ramBytes) / (1024.0 * 1024.0),
		})
		return true
	})

	c.JSON(200, gin.H{
		"servers":         servers,
		"number_of_servers": len(servers),
	})
}

func handleSummaryEndpoint(c *gin.Context, cache *bfrest.BigFixCache) {
	if !requireAuth(c) {
		return
	}
	
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
			itemSize := int64(len(v.Json))
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
		serverSummary["ram_bytes"] = serverSize
		serverSummary["ram_kb"] = float64(serverSize) / 1024.0
		serverSummary["ram_mb"] = float64(serverSize) / (1024.0 * 1024.0)
		summary[server.ServerName] = serverSummary
		totalSize += serverSize

		return true
	})

	summary["total_ram_bytes"] = totalSize
	summary["total_ram_kb"] = float64(totalSize) / 1024.0
	summary["total_ram_mb"] = float64(totalSize) / (1024.0 * 1024.0)
	c.JSON(200, summary)
}

func handleCacheEndpoint(c *gin.Context, cache *bfrest.BigFixCache) {
	if !requireAuth(c) {
		return
	}
	
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
}