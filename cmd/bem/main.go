package main

import (
	"encoding/xml"
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

	result, err := bfrest.Get("https://10.10.220.60:52311/api/computers", "IEMAdmin", "BigFix!123")

	fmt.Println(result)
	fmt.Println(err)

	var api bfrest.BESAPI
	err = xml.Unmarshal(([]byte)(result.RawXML), &api)
	if err != nil {
		fmt.Println("Error parsing XML:", err)
		return
	}

	fmt.Println(api)

}
