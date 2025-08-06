// Package example0 provides an integration test for when wasinet is not involved
package main

import (
	"log"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.Println("broken")
}
