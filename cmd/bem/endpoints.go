package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

// HTTP endpoint handlers

func setupRoutes(r *gin.Engine, cache *bfrest.BigFixCache, config Config) {
	// OTP endpoint for admin session creation (no authentication required)
	r.GET("/otp", handleOTPEndpoint)

	// Client registration endpoint (no authentication required)
	r.POST("/register", func(c *gin.Context) {
		handleRegisterEndpoint(c, config)
	})

	// Registration request endpoint (no authentication required)
	r.GET("/requestregistration", func(c *gin.Context) {
		handleRegistrationRequest(c, config)
	})

	// Help endpoint (no authentication required)
	r.GET("/help", handleHelpEndpoint)

	// Protected endpoints require authentication
	r.GET("/urls", func(c *gin.Context) {
		handleURLsEndpoint(c, cache)
	})

	r.GET("/servers", func(c *gin.Context) {
		handleServersEndpoint(c, cache)
	})

	r.GET("/summary", func(c *gin.Context) {
		handleSummaryEndpoint(c, cache)
	})

	r.GET("/cache", func(c *gin.Context) {
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
}

func handleServersEndpoint(c *gin.Context, cache *bfrest.BigFixCache) {
	if !requireAuth(c) {
		return
	}
	
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