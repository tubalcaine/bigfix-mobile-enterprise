package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *slog.Logger
var ginLogWriter io.Writer

// InitLogger sets up the global logger with optional file rotation
func InitLogger(config Config) error {
	var level slog.Level

	if config.Debug != 0 {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	// Create a handler with custom options
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.Debug != 0,
	}

	// Determine output destination(s)
	var writer io.Writer

	if config.LogToFile {
		// Set default log file path if not specified
		logFilePath := config.LogFilePath
		if logFilePath == "" {
			logFilePath = "./logs/bem.log"
		}

		// Ensure log directory exists
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Set default values for rotation parameters
		maxSizeMB := config.LogMaxSizeMB
		if maxSizeMB == 0 {
			maxSizeMB = 100 // 100MB default
		}
		maxBackups := config.LogMaxBackups
		if maxBackups == 0 {
			maxBackups = 5
		}
		maxAgeDays := config.LogMaxAgeDays
		if maxAgeDays == 0 {
			maxAgeDays = 30
		}

		// Configure lumberjack for log rotation
		fileWriter := &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    maxSizeMB,    // megabytes
			MaxBackups: maxBackups,   // number of backups
			MaxAge:     maxAgeDays,   // days
			Compress:   config.LogCompress,
		}

		// Combine console + file if both enabled
		if config.LogToConsole {
			writer = io.MultiWriter(os.Stdout, fileWriter)
		} else {
			writer = fileWriter
		}
	} else {
		// Console only (default behavior)
		writer = os.Stdout
	}

	// Create handler with chosen writer
	handler := slog.NewTextHandler(writer, handlerOpts)
	logger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	// Set up Gin log writer (same destination as slog)
	ginLogWriter = writer

	logger.Info("Logger initialized",
		"level", level.String(),
		"debug_mode", config.Debug != 0,
		"log_to_file", config.LogToFile,
		"log_file_path", config.LogFilePath,
		"log_to_console", config.LogToConsole,
	)

	return nil
}

// GetLogger returns the configured logger instance
func GetLogger() *slog.Logger {
	if logger == nil {
		// Fallback to default if not initialized
		return slog.Default()
	}
	return logger
}

// GetGinLogWriter returns the writer for Gin logging
func GetGinLogWriter() io.Writer {
	if ginLogWriter == nil {
		return os.Stdout
	}
	return ginLogWriter
}
