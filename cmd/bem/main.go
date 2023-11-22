package main

import (
	"fmt"

	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

const (
	app_version = "0.0"
	app_name    = "bem"
	app_desc    = "BigFix Enterprise Mobile Server"
)

func main() {
	fmt.Println(app_desc)
	fmt.Println("Version " + app_version)

	go bfrest.PopulateCoreTypes("https://10.10.220.60:52311", "IEMAdmin", "BigFix!123")

	// At this point we will start a web service, but for now, just loop
	// and wait for input so the program doesn't exit.
	for {
		fmt.Println("Enter a query:")
		var query string
		fmt.Scanln(&query)
		if query == "exit" {
			break
		}
	}

	fmt.Println(bfrest.GetCache())
}
