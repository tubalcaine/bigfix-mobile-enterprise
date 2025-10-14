package main

import (
	"sync"
	"time"
)

// Application metadata
var (
	app_version = "0.1"
	app_desc    = "BigFix Enterprise Mobile Server"
)

// Configuration structures
type Config struct {
	AppCacheTimeout          uint64         `json:"app_cache_timeout"`
	BigFixServers            []BigFixServer `json:"bigfix_servers"`
	ListenPort               int            `json:"listen_port"`
	CertPath                 string         `json:"cert_path"`
	KeyPath                  string         `json:"key_path"`
	KeySize                  int            `json:"keysize"`
	RegistrationDir          string         `json:"registration_dir"`
	RequestsDir              string         `json:"requests_dir"`
	RegistrationDataDir      string         `json:"registration_data_dir"`
	GarbageCollectorInterval uint64         `json:"garbage_collector_interval"` // seconds between GC sweeps, default 15
	MaxCacheLifetime         uint64         `json:"max_cache_lifetime"`          // maximum cache lifetime in seconds, default 86400 (24 hours)
}

type BigFixServer struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	MaxAge   uint64 `json:"maxage"`
	PoolSize int    `json:"poolsize"`
}

// Registration and client management structures
type RegistrationOTP struct {
	ClientName      string    `json:"client_name"`
	OneTimeKey      string    `json:"one_time_key"`
	KeyLifespanDays int       `json:"key_lifespan_days,omitempty"` // 0 = never expires
	CreatedAt       time.Time `json:"created_at"`
	RequestedBy     string    `json:"requested_by,omitempty"`
}

type RegisteredClient struct {
	ClientName      string     `json:"client_name"`
	PublicKey       string     `json:"public_key"` // PEM-encoded for JSON storage
	RegisteredAt    time.Time  `json:"registered_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"` // nil if never expires
	LastUsed        time.Time  `json:"last_used,omitempty"`
	KeyLifespanDays int        `json:"key_lifespan_days"`
}

// API request/response structures
type RegisterRequest struct {
	ClientName string `json:"client_name"`
	OneTimeKey string `json:"one_time_key"`
}

type RegisterResponse struct {
	Success    bool   `json:"success"`
	PrivateKey string `json:"private_key,omitempty"` // PEM-encoded private key
	Message    string `json:"message,omitempty"`
}

// Global state variables
var (
	// Global state for client registration
	registrationOTPs      []RegistrationOTP
	registeredClients     []RegisteredClient
	registrationMutex     sync.RWMutex
	configDir            string
	registrationDataDir  string
	
	// Session management for cookie-based admin access
	activeSessions        map[string]time.Time // sessionToken -> expiresAt
	sessionMutex          sync.RWMutex
)