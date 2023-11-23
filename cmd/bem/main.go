package main

import (
	"fmt"

	"github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)

const (
	app_version = "0.0"
	app_name    = "bem"
	app_desc    = "BigFix Enterprise Mobile Server"
	app_user    = "IEMAdmin"
	app_pass    = "BigFix!123"
)

func main() {
	fmt.Println(app_desc)
	fmt.Println("Version " + app_version)

	go bfrest.PopulateCoreTypes("https://10.10.220.60:52311", app_user, app_pass)
	go bfrest.PopulateCoreTypes("https://10.10.220.59:52311", app_user, app_pass)

	cache := bfrest.GetCache()

	// At this point we will start a web service, but for now, just loop
	// and wait for input so the program doesn't exit.
	for {
		fmt.Println("\n\nEnter a url (exit to terminate): ")
		var query string
		fmt.Scanln(&query)
		if query == "exit" {
			break
		}

		if query == "cache" {
			cache.ServerCache.Range(func(key, value interface{}) bool {
				server := value.(*bfrest.BigFixServerCache)
				fmt.Println(server.ServerName)
				server.CacheMap.Range(func(key, value interface{}) bool {
					fmt.Printf("\t%s\n", key.(string))
					return true
				})

				return true
			})
			continue
		}

		fmt.Println(cache.Get(query, app_user, app_pass))
	}
}
