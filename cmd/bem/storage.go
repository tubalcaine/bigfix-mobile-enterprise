package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Storage functions for persistent client registration data

func createBackup(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil // No file to backup
	}
	
	// Find next backup number
	backupNum := 1
	for {
		backupName := fmt.Sprintf("%s.bak.%d", filename, backupNum)
		if _, err := os.Stat(backupName); os.IsNotExist(err) {
			return os.Rename(filename, backupName)
		}
		backupNum++
	}
}

func saveRegistrationOTPs() error {
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(registrationDataDir, 0700); err != nil {
		return fmt.Errorf("failed to create registration data directory: %v", err)
	}
	
	filename := filepath.Join(registrationDataDir, "registration_otps.json")
	
	// Create backup
	if err := createBackup(filename); err != nil {
		slog.Warn("Could not create backup", "filename", filename, "error", err)
	}
	
	// Write to temporary file first, then rename (atomic operation)
	tmpFile := filename + ".tmp"
	data, err := json.MarshalIndent(registrationOTPs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registration OTPs: %v", err)
	}
	
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write registration OTPs: %v", err)
	}
	
	return os.Rename(tmpFile, filename)
}

func loadRegistrationOTPs() error {
	filename := filepath.Join(registrationDataDir, "registration_otps.json")
	
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		registrationOTPs = make([]RegistrationOTP, 0)
		return nil // File doesn't exist yet, start with empty slice
	}
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read registration OTPs: %v", err)
	}
	
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	
	if err := json.Unmarshal(data, &registrationOTPs); err != nil {
		return fmt.Errorf("failed to parse registration OTPs: %v", err)
	}
	
	return nil
}

// saveRegisteredClientsUnlocked saves without acquiring mutex (caller must hold lock)
func saveRegisteredClientsUnlocked() error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(registrationDataDir, 0700); err != nil {
		return fmt.Errorf("failed to create registration data directory: %v", err)
	}
	
	filename := filepath.Join(registrationDataDir, "registered_clients.json")
	
	// Create backup
	if err := createBackup(filename); err != nil {
		slog.Warn("Could not create backup", "filename", filename, "error", err)
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(registeredClients, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registered clients: %v", err)
	}
	
	// Write to file with restricted permissions
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write registered clients file: %v", err)
	}
	
	return nil
}

func saveRegisteredClients() error {
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	
	return saveRegisteredClientsUnlocked()
}

func loadRegisteredClients() error {
	filename := filepath.Join(registrationDataDir, "registered_clients.json")
	
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		registeredClients = make([]RegisteredClient, 0)
		return nil // File doesn't exist yet, start with empty slice
	}
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read registered clients: %v", err)
	}
	
	registrationMutex.Lock()
	defer registrationMutex.Unlock()
	
	if err := json.Unmarshal(data, &registeredClients); err != nil {
		return fmt.Errorf("failed to parse registered clients: %v", err)
	}
	
	// Validate public keys and remove expired clients
	validClients := make([]RegisteredClient, 0)
	for _, client := range registeredClients {
		// Validate PEM-encoded public key
		block, _ := pem.Decode([]byte(client.PublicKey))
		if block == nil {
			slog.Warn("Invalid PEM key for client, removing", "client_name", client.ClientName)
			continue
		}

		_, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			slog.Warn("Invalid public key for client, removing", "client_name", client.ClientName, "error", err)
			continue
		}

		// Check if expired
		if client.ExpiresAt != nil && time.Now().After(*client.ExpiresAt) {
			slog.Info("Expired client removed", "client_name", client.ClientName)
			continue
		}
		
		validClients = append(validClients, client)
	}
	
	// Update slice and save if any clients were removed
	if len(validClients) != len(registeredClients) {
		registeredClients = validClients
		return saveRegisteredClients() // This will create a backup of the cleaned version
	}
	
	return nil
}