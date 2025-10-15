package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Request structures
type RegistrationRequest struct {
	ClientName string `json:"client_name"`
}

type RegistrationRequestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

// Safe filename generation - prevent directory traversal and other attacks
func sanitizeFilename(clientName string) string {
	// Remove or replace unsafe characters
	// Allow only alphanumeric, hyphens, underscores, and periods
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	safe := reg.ReplaceAllString(clientName, "_")
	
	// Prevent directory traversal
	safe = strings.ReplaceAll(safe, "..", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	
	// Ensure it's not empty and not too long
	if safe == "" {
		safe = "unnamed_client"
	}
	if len(safe) > 100 {
		safe = safe[:100]
	}
	
	// Ensure it doesn't start with a dot (hidden file)
	if strings.HasPrefix(safe, ".") {
		safe = "client_" + safe[1:]
	}
	
	return safe
}

// Create registration request file
func createRegistrationRequestFile(clientName, requestsDir string) (string, error) {
	if requestsDir == "" {
		return "", fmt.Errorf("requests directory not configured")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create requests directory: %v", err)
	}
	
	// Generate safe filename
	safeFilename := sanitizeFilename(clientName)
	filename := filepath.Join(requestsDir, safeFilename+".json")
	
	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return filename, fmt.Errorf("registration request already exists for client: %s", clientName)
	}
	
	// Create the request file in the format that can be moved to registration_dir
	requestData := []RegistrationOTP{
		{
			ClientName:      clientName,
			OneTimeKey:      "EDIT_THIS_VALUE", // Admin will replace this
			KeyLifespanDays: 365,               // Default to 1 year
			CreatedAt:       time.Now(),
			RequestedBy:     "client-request",
		},
	}
	
	// Write the file
	data, err := json.MarshalIndent(requestData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %v", err)
	}
	
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write request file: %v", err)
	}
	
	return filename, nil
}

// HTTP handler for registration requests
func handleRegistrationRequest(c *gin.Context, config Config) {
	clientName := c.Query("ClientName")
	if clientName == "" {
		c.JSON(400, RegistrationRequestResponse{
			Success: false,
			Message: "ClientName parameter is required",
		})
		return
	}
	
	// Validate ClientName length and basic format
	if len(clientName) < 1 || len(clientName) > 100 {
		c.JSON(400, RegistrationRequestResponse{
			Success: false,
			Message: "ClientName must be between 1 and 100 characters",
		})
		return
	}
	
	// Create the request file
	filename, err := createRegistrationRequestFile(clientName, config.RequestsDir)
	if err != nil {
		log.Printf("Failed to create registration request for %s: %v", clientName, err)

		// Check if this is a "file already exists" error
		if strings.Contains(err.Error(), "registration request already exists") {
			c.JSON(409, RegistrationRequestResponse{
				Success: false,
				Message: err.Error(),
			})
		} else {
			c.JSON(500, RegistrationRequestResponse{
				Success: false,
				Message: err.Error(),
			})
		}
		return
	}

	log.Printf("Registration request created for client: %s (file: %s)", clientName, filename)
	c.JSON(200, RegistrationRequestResponse{
		Success: true,
		Message: fmt.Sprintf("Registration request created successfully. Admin must edit the OneTimeKey in the file and move it to the registration directory."),
		File:    filepath.Base(filename),
	})
}