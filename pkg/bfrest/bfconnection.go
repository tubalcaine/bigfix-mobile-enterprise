// NewBFConnection creates and initializes a new BFConnection instance.
package bfrest

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

// BFConnection represents a connection configuration.
type BFConnection struct {
	URL      string
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
func NewConnectionPool(url string, username string, password string) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		connections: make([]*BFConnection, 0),
		channel:     make(chan *BFConnection),
	}

	for i := 0; i < 5; i++ {
		connection, err := createBFConnection(url, username, password)

		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		pool.connections = append(pool.connections, connection)
		pool.channel <- connection
	}

	return pool, nil
}

// createBFConnection creates a new BFConnection instance.
func createBFConnection(urlStr string, username string, password string) (*BFConnection, error) {
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
		URL:      urlStr,
		Username: username,
		Password: password,
		Conn:     client,
		tr:       &transport,
	}, nil
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
