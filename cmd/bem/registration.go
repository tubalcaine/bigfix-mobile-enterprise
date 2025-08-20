package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Registration directory monitoring functions

func processRegistrationFile(filename string) {
	log.Printf("Processing registration file: %s", filename)
	
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading registration file %s: %v", filename, err)
		return
	}
	
	// Parse JSON array of registration OTPs
	var newOTPs []RegistrationOTP
	if err := json.Unmarshal(data, &newOTPs); err != nil {
		log.Printf("Error parsing registration file %s: %v", filename, err)
		return
	}
	
	// Add CreatedAt timestamp to new OTPs
	now := time.Now()
	for i := range newOTPs {
		newOTPs[i].CreatedAt = now
	}
	
	// Add to our slice and save
	registrationMutex.Lock()
	registrationOTPs = append(registrationOTPs, newOTPs...)
	registrationMutex.Unlock()
	
	if err := saveRegistrationOTPs(); err != nil {
		log.Printf("Error saving registration OTPs: %v", err)
		return
	}
	
	// Remove the processed file
	if err := os.Remove(filename); err != nil {
		log.Printf("Warning: Could not remove processed registration file %s: %v", filename, err)
	} else {
		log.Printf("Successfully processed and removed registration file %s, added %d OTPs", filename, len(newOTPs))
	}
}

func watchRegistrationDirectory(dir string) {
	if dir == "" {
		log.Println("No registration directory configured, skipping file monitoring")
		return
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Printf("Error creating registration directory %s: %v", dir, err)
		return
	}
	
	// Process any existing files first
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Error reading registration directory %s: %v", dir, err)
		return
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			fullPath := filepath.Join(dir, entry.Name())
			processRegistrationFile(fullPath)
		}
	}
	
	// Set up filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Error creating filesystem watcher: %v", err)
		return
	}
	defer watcher.Close()
	
	// Start monitoring goroutine
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				
				// Only process JSON files that are created or written
				if (event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write) &&
					strings.HasSuffix(strings.ToLower(event.Name), ".json") {
					
					// Small delay to ensure file write is complete
					time.Sleep(100 * time.Millisecond)
					processRegistrationFile(event.Name)
				}
				
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Filesystem watcher error: %v", err)
			}
		}
	}()
	
	// Add the directory to watch
	err = watcher.Add(dir)
	if err != nil {
		log.Printf("Error watching registration directory %s: %v", dir, err)
		return
	}
	
	log.Printf("Watching registration directory: %s", dir)
}

// Client registration and authentication functions

func isClientRegistered(clientName string) bool {
	registrationMutex.RLock()
	defer registrationMutex.RUnlock()
	
	for _, client := range registeredClients {
		if client.ClientName == clientName {
			// Check if expired
			if client.ExpiresAt != nil && time.Now().After(*client.ExpiresAt) {
				return false // Expired
			}
			return true
		}
	}
	return false
}

func findAndRemoveOTP(clientName, oneTimeKey string) (*RegistrationOTP, bool) {
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	
	for i, otp := range registrationOTPs {
		if otp.ClientName == clientName && otp.OneTimeKey == oneTimeKey {
			// Remove from slice
			registrationOTPs = append(registrationOTPs[:i], registrationOTPs[i+1:]...)
			return &otp, true
		}
	}
	return nil, false
}

func findAndRemoveOTPByKey(oneTimeKey string) (*RegistrationOTP, bool) {
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	
	for i, otp := range registrationOTPs {
		if otp.OneTimeKey == oneTimeKey {
			// Remove from slice
			registrationOTPs = append(registrationOTPs[:i], registrationOTPs[i+1:]...)
			return &otp, true
		}
	}
	return nil, false
}

func generateAndRegisterClient(otp RegistrationOTP, keySize int) (*RegisterResponse, error) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %v", err)
	}
	
	// Encode private key as PEM for client
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)
	
	// Encode public key as PEM for storage
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %v", err)
	}
	
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyString := string(pem.EncodeToMemory(publicKeyPEM))
	
	// Calculate expiration date
	var expiresAt *time.Time
	if otp.KeyLifespanDays > 0 {
		expiry := time.Now().AddDate(0, 0, otp.KeyLifespanDays)
		expiresAt = &expiry
	}
	
	// Create registered client record
	client := RegisteredClient{
		ClientName:      otp.ClientName,
		PublicKey:       publicKeyString,
		RegisteredAt:    time.Now(),
		ExpiresAt:       expiresAt,
		LastUsed:        time.Now(),
		KeyLifespanDays: otp.KeyLifespanDays,
	}
	
	// Add to registered clients slice
	registrationMutex.Lock()
	registeredClients = append(registeredClients, client)
	registrationMutex.Unlock()
	
	// Save to disk
	if err := saveRegisteredClients(); err != nil {
		return nil, fmt.Errorf("failed to save registered clients: %v", err)
	}
	
	// Save updated OTPs (with the used one removed)
	if err := saveRegistrationOTPs(); err != nil {
		return nil, fmt.Errorf("failed to save registration OTPs: %v", err)
	}
	
	return &RegisterResponse{
		Success:    true,
		PrivateKey: string(privateKeyBytes),
		Message:    "Client registered successfully",
	}, nil
}