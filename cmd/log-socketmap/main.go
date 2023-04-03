// Command log-socketmap is a socketmap server that always returns NOTFOUND and logs all requests.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/d--j/go-socketmap"
)

func main() {
	var protocol, address string
	flag.StringVar(&protocol,
		"proto",
		"tcp",
		"Protocol family (`unix or tcp`)")
	flag.StringVar(&address,
		"addr",
		"127.0.0.1:10931",
		"Bind to address/port or unix domain socket path")
	flag.Parse()
	if protocol != "unix" && protocol != "tcp" {
		log.Fatalf("invalid protocol name %q", protocol)
	}
	err := socketmap.ListenAndServe(protocol, address, func(_ context.Context, lookup, key string) (result string, found bool, err error) {
		log.Printf("%s %s", lookup, key)
		return "", false, nil
	})
	log.Println(err)
}
