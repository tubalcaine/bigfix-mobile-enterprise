package main


import (
	"fmt"
	"io/ioutil"
	"net/http"
	"crypto/tls"
	"encoding/xml"
	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

const (
	app_version = "0.0"
	app_name = "bem"
	app_desc = "BigFix Enterprise Mobile Server"
)

// getDataFromAPI makes a GET request to the specified URL with HTTP Basic Authentication
// and returns the raw data payload and an error if any.
func getDataFromAPI(url, username, password string) ([]byte, error) {
	// Skip certificate verification (use with caution)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set the username and password for HTTP Basic Authentication
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-200 response status: %d - %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	return data, nil
}

func main() {
	fmt.Println(app_desc)
	fmt.Println("Version " + app_version)

	url := "https://10.10.220.60:52311/api/computers" // Replace with your actual URL
	username := "IEMAdmin"               // Replace with your actual username
	password := "BigFix!123"               // Replace with your actual password

	data, err := getDataFromAPI(url, username, password)
	if err != nil {
		fmt.Printf("An error occurred: %v\n", err)
		return
	}

	// Data contains the raw XML payload.
	fmt.Printf("Raw XML Data: %s\n", string(data))

	// Unmarshal the XML data into Go structures
	var computers bfrest.BESAPI
	err = xml.Unmarshal(data, &computers)
	if err != nil {
		fmt.Printf("Error unmarshaling XML data: %v\n", err)
		return
	}

	// Print the first computer name
	if len(computers.Computer) > 0 {
		fmt.Printf("First computer name: %d\n", computers.Computer[0].ID)
	}
}
