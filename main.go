package main

import (
	"log"
	"net"
	"time"
)

func main() {
	for {
		ips, err := net.LookupIP("grpc-cluster.")
		if err != nil {
			log.Fatalf("Cannot look up hostname: %v", err)
		}
		log.Printf("Found IPs: %v", ips)
		time.Sleep(5 * time.Second)
	}
}
