package main

import (
	"sync"
	"time"
)

// Application metadata
var (
	app_desc = "BigFix Enterprise Mobile Server"
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
	Debug                    int            `json:"debug"`                       // DEPRECATED: use log_level instead. 0 = debug logging off, non-zero = debug logging on

	// Logging configuration
	LogLevel      string `json:"log_level"`         // log level: "DEBUG", "INFO", "WARN", or "ERROR" (default: "INFO", or "DEBUG" if debug=1)
	LogToFile     bool   `json:"log_to_file"`       // enable file logging
	LogFilePath   string `json:"log_file_path"`     // path to log file
	LogMaxSizeMB  int    `json:"log_max_size_mb"`   // maximum size in megabytes before rotation
	LogMaxBackups int    `json:"log_max_backups"`   // maximum number of old log files to retain
	LogMaxAgeDays int    `json:"log_max_age_days"`  // maximum number of days to retain old log files
	LogCompress   bool   `json:"log_compress"`      // compress old log files with gzip
	LogToConsole  bool   `json:"log_to_console"`    // also log to stdout (in addition to file)
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
	// Global configuration
	appConfig             *Config

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