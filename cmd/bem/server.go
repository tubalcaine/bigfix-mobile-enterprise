package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// TLSListener wraps a net.Listener to log TLS connection details and errors
type TLSListener struct {
	net.Listener
	logger *slog.Logger
}

// Accept wraps the Accept method to log connection details
func (l *TLSListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// Log the incoming connection
	remoteAddr := conn.RemoteAddr().String()
	l.logger.Debug("New connection accepted", "remote_addr", remoteAddr)

	// Wrap the connection to capture TLS handshake details
	return &loggingConn{
		Conn:   conn,
		logger: l.logger,
		remote: remoteAddr,
	}, nil
}

// loggingConn wraps net.Conn to log TLS handshake details
type loggingConn struct {
	net.Conn
	logger      *slog.Logger
	remote      string
	handshakeDone bool
}

// Read wraps the Read method to capture TLS errors
func (c *loggingConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	// Log TLS handshake completion on first successful read
	if !c.handshakeDone && n > 0 {
		if tlsConn, ok := c.Conn.(*tls.Conn); ok {
			state := tlsConn.ConnectionState()
			if state.HandshakeComplete {
				c.handshakeDone = true
				c.logger.Debug("TLS handshake completed",
					"remote_addr", c.remote,
					"tls_version", tlsVersionString(state.Version),
					"cipher_suite", tls.CipherSuiteName(state.CipherSuite),
					"server_name", state.ServerName,
				)
			}
		}
	}

	if err != nil && !c.handshakeDone {
		// Log TLS errors that occur before handshake completion
		c.logger.Error("TLS connection error (likely handshake failure)",
			"error", err,
			"remote_addr", c.remote,
			"bytes_read", n,
		)
	}

	return n, err
}

// tlsVersionString converts TLS version constant to string
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// StartTLSServer starts the HTTP server with TLS and comprehensive logging
func StartTLSServer(handler http.Handler, certPath, keyPath string, port int, logger *slog.Logger) error {
	// Load TLS certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	// Create base listener
	addr := fmt.Sprintf(":%d", port)
	baseListener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Wrap with TLS
	tlsListener := tls.NewListener(baseListener, tlsConfig)

	// Wrap with logging listener
	loggingListener := &TLSListener{
		Listener: tlsListener,
		logger:   logger,
	}

	// Create HTTP server
	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("Starting TLS server",
		"port", port,
		"min_tls_version", "TLS 1.2",
	)

	// Serve with custom listener
	return server.Serve(loggingListener)
}
