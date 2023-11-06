package main

import "fmt"

const (
	app_version := "0.0"
	app_name := "bem"
	app_desc := "BigFix Enterprise Mobile Server"
)

func main() {
	fmt.Println(app_desc)
	fmt.Println("Version " + app_version)
}