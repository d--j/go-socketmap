// Command test-socketmap is a test client for a socketmap server
package main

import (
	"flag"
	"log"
	"os"

	"github.com/d--j/go-socketmap"
)

func main() {
	var protocol, address, lookup, key string
	flag.StringVar(&protocol,
		"proto",
		"tcp",
		"Protocol family (`unix or tcp`)")
	flag.StringVar(&address,
		"addr",
		"127.0.0.1:10931",
		"Bind to address/port or unix domain socket path")
	flag.StringVar(&lookup,
		"map",
		"",
		"`name` of the map to use for lookup")
	flag.StringVar(&key,
		"key",
		"",
		"`name` of the entry to lookup")
	flag.Parse()
	if protocol != "unix" && protocol != "tcp" {
		log.Fatalf("invalid protocol name %q", protocol)
	}
	if lookup == "" || key == "" {
		flag.Usage()
		os.Exit(1)
	}
	c := socketmap.NewClient(protocol, address)
	result, found, err := c.Lookup(lookup, key)
	c.Close()
	if err != nil {
		log.Fatal(err)
	}
	if found {
		log.Printf("OK %q", result)
	} else {
		log.Printf("NOTFOUND")
	}
}
