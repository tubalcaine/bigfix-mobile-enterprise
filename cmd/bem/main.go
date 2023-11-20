package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

const (
	app_version = "0.0"
	app_name    = "bem"
	app_desc    = "BigFix Enterprise Mobile Server"
)

// getDataFromAPI makes a GET request to the specified URL with HTTP Basic Authentication
// and returns the raw data payload and an error if any.
func getDataFromAPI(conn *bfrest.BFConnection, resource string) ([]byte, error) {
	req, err := http.NewRequest("GET", conn.URL+resource, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set the username and password for HTTP Basic Authentication
	req.SetBasicAuth(conn.Username, conn.Password)

	resp, err := conn.Conn.Do(req)

	if err != nil {
		return nil, fmt.Errorf("error making GET request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-200 response status: %d - %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	return data, nil
}

func main() {
	fmt.Println(app_desc)
	fmt.Println("Version " + app_version)

	cpool, _ := bfrest.NewPool("https://10.10.220.60:52311", "IEMAdmin", "BigFix!123", 5)

	fmt.Printf("Connection pool has %d items\n", cpool.Len())

	conn, _ := cpool.Acquire()

	fmt.Printf("Connection pool has %d items\n", cpool.Len())

	url := "/api/computers" // Replace with your actual URL

	data, err := getDataFromAPI(conn, url)

	if err != nil {
		fmt.Printf("An error occurred: %v\n", err)
		return
	}

	// Data contains the raw XML payload.
	fmt.Printf("Raw XML Data: %s\n", string(data))

	cpool.Release(conn)

	// Unmarshal the XML data into Go structures
	var computers bfrest.BESAPI
	err = xml.Unmarshal(data, &computers)
	if err != nil {
		fmt.Printf("Error unmarshaling XML data: %v\n", err)
		return
	}

	for _, computer := range computers.Computer {
		fmt.Printf("Computer: %d\n", computer.ID)
	}
}
