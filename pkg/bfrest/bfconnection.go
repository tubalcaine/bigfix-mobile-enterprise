// NewBFConnection creates and initializes a new BFConnection instance.
package bfrest

import (
	"crypto/tls"
	"net/http"
)

// BFConnection represents a connection configuration.
type BFConnection struct {
	URL      string
	Port     int
	Username string
	Password string
	Conn     http.Client
	tr       *http.Transport
}

// ConnectionPool represents a pool of BFConnections.
type ConnectionPool struct {
	connections []*BFConnection
	channel     chan *BFConnection
}

// NewConnectionPool creates and initializes a new ConnectionPool instance.
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		connections: make([]*BFConnection, 0),
		channel:     make(chan *BFConnection),
	}

	// Initialize the pool with 5 BFConnections.
	for i := 0; i < 5; i++ {
		connection := createBFConnection()
		pool.connections = append(pool.connections, connection)
		pool.channel <- connection
	}

	return pool
}

// createBFConnection creates a new BFConnection instance.
func createBFConnection() *BFConnection {
	// Initialize the http.Transport. You might want to customize this based on your requirements.
	transport := http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Initialize the http.Client. You can also customize this as needed.
	client := http.Client{
		Transport: &transport,
	}

	// Return a new BFConnection with the provided details.
	return &BFConnection{
		URL:      "",
		Port:     0,
		Username: "",
		Password: "",
		Conn:     client,
		tr:       &transport,
	}
}

// GetAvailableConnections returns the number of available BFConnections.
func (pool *ConnectionPool) GetAvailableConnections() int {
	return len(pool.connections)
}

// PopConnection pops a BFConnection from the pool.
func (pool *ConnectionPool) PopConnection() *BFConnection {
	connection := <-pool.channel
	return connection
}

// PushConnection pushes a BFConnection back to the pool.
func (pool *ConnectionPool) PushConnection(connection *BFConnection) {
	pool.channel <- connection
}
