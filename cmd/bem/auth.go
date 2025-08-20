package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"strings"
	"time"
	
	"github.com/gin-gonic/gin"
)

// Session management functions for cookie-based authentication

func generateSessionToken() string {
	// Generate a secure random session token
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func createAdminSession(otp RegistrationOTP) string {
	sessionToken := generateSessionToken()
	expiresAt := time.Now().Add(8 * time.Hour) // 8-hour working day
	
	sessionMutex.Lock()
	if activeSessions == nil {
		activeSessions = make(map[string]time.Time)
	}
	activeSessions[sessionToken] = expiresAt
	sessionMutex.Unlock()
	
	log.Printf("Created admin session for %s (expires at %s)", otp.ClientName, expiresAt.Format("15:04:05"))
	return sessionToken
}

func isValidSession(sessionToken string) bool {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	
	if activeSessions == nil {
		return false
	}
	
	expiresAt, exists := activeSessions[sessionToken]
	if !exists {
		return false
	}
	
	// Check if session has expired
	if time.Now().After(expiresAt) {
		// Clean up expired session (do this outside the read lock)
		go func() {
			sessionMutex.Lock()
			delete(activeSessions, sessionToken)
			sessionMutex.Unlock()
		}()
		return false
	}
	
	return true
}

func cleanupExpiredSessions() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	
	if activeSessions == nil {
		return
	}
	
	now := time.Now()
	for token, expiresAt := range activeSessions {
		if now.After(expiresAt) {
			delete(activeSessions, token)
		}
	}
}

// Client key validation functions

func isValidClientKey(encodedPrivateKey string) (string, bool) {
	// Decode base64 private key
	privateKeyBytes, err := base64.StdEncoding.DecodeString(encodedPrivateKey)
	if err != nil {
		log.Printf("Failed to decode client key: %v", err)
		return "", false
	}
	
	// Parse PEM-encoded private key
	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		log.Printf("Failed to parse PEM block from client key")
		return "", false
	}
	
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Printf("Failed to parse RSA private key: %v", err)
		return "", false
	}
	
	// Derive public key from private key
	publicKey := &privateKey.PublicKey
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Printf("Failed to marshal public key: %v", err)
		return "", false
	}
	
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyString := string(pem.EncodeToMemory(publicKeyPEM))
	
	// Check if this public key matches any registered client
	registrationMutex.RLock()
	defer registrationMutex.RUnlock()
	
	for _, client := range registeredClients {
		if client.PublicKey == publicKeyString {
			// Check if expired
			if client.ExpiresAt != nil && time.Now().After(*client.ExpiresAt) {
				log.Printf("Client %s key has expired", client.ClientName)
				return "", false
			}
			
			// Update last used time
			go func(clientName string) {
				registrationMutex.Lock()
				defer registrationMutex.Unlock()
				for i := range registeredClients {
					if registeredClients[i].ClientName == clientName {
						registeredClients[i].LastUsed = time.Now()
						break
					}
				}
				saveRegisteredClients() // Update persistent storage
			}(client.ClientName)
			
			return client.ClientName, true
		}
	}
	
	log.Printf("No matching registered client found for provided key")
	return "", false
}

func isAuthenticatedRequest(c *gin.Context) bool {
	// Check for valid session cookie (admin access)
	cookie, err := c.Cookie("bem_session")
	if err == nil && isValidSession(cookie) {
		return true
	}
	
	// Check for client key authentication via Authorization header
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Client ") {
		clientKey := strings.TrimPrefix(authHeader, "Client ")
		clientName, valid := isValidClientKey(clientKey)
		if valid {
			// Store client name in context for logging/debugging
			c.Set("client_name", clientName)
			return true
		}
		log.Printf("Invalid client key authentication attempt")
	}
	
	return false
}

// Authentication middleware helper
func requireAuth(c *gin.Context) bool {
	if !isAuthenticatedRequest(c) {
		// Check if this was a client key attempt that failed due to expiration
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Client ") {
			c.JSON(401, gin.H{
				"error":   "Client authentication failed. Key may be expired or invalid.",
				"expired": true, // Signal to Android app to discard and re-register
			})
		} else {
			c.JSON(401, gin.H{
				"error": "Authentication required. Please visit /otp?OneTimeKey=<key> or register your client.",
			})
		}
		return false
	}
	return true
}