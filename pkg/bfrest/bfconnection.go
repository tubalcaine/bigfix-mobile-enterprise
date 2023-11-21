package bfrest

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// BFConnection represents a connection configuration.
type BFConnection struct {
	URL      string
	Username string
	Password string
	Conn     http.Client
}

// Get sends a GET request to the specified URL and returns the response body as a string.

func (c *BFConnection) Get(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Compare the non-directory components of the BFConnection URL and the parsed URL
	if c.URL != parsedURL.Scheme+"://"+parsedURL.Host {
		return "", fmt.Errorf("URL does not match")
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.Conn.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
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
	}, nil
}

// Pool manages a set of connections.
type Pool struct {
	connections chan *BFConnection
	factory     func() (*BFConnection, error)
	closed      bool
	mutex       sync.Mutex
}

// NewPool creates a new pool of connections.
func NewPool(urlStr, username, password string, size int) (*Pool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("size value too small")
	}

	factory := func() (*BFConnection, error) {
		return createBFConnection(urlStr, username, password)
	}

	pool := &Pool{
		connections: make(chan *BFConnection, size),
		factory:     factory,
	}

	for i := 0; i < size; i++ {
		connection, err := factory()
		if err != nil {
			return nil, err
		}
		pool.connections <- connection
	}

	return pool, nil
}

// Return number of connections in the pool.
func (p *Pool) Len() int {
	return len(p.connections)
}

// Acquire retrieves a connection from the pool.
func (p *Pool) Acquire() (*BFConnection, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	fmt.Println("Acquire")
	if p.closed {
		return nil, fmt.Errorf("pool is closed")
	}

	return <-p.connections, nil
}

// Release returns a connection to the pool.
func (p *Pool) Release(c *BFConnection) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	fmt.Println("Release")
	if p.closed {
		// handle closed pool scenario, maybe discard the connection
		return
	}

	p.connections <- c
}

// Close closes the pool and releases all connections.
func (p *Pool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.connections)
	for r := range p.connections {
		// Close or cleanup the resource.
		r.Conn.CloseIdleConnections()
	}
}
