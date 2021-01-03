package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"time"
)

func main() {
	env := os.Environ()
	for _, item := range env {
		log.Printf("Environment: %s", item)
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if len(namespace) == 0 {
		log.Fatal("Environment variable POD_NAMESPACE is not set.")
	}

	targetname := fmt.Sprintf("grpc-cluster.%s.svc.cluster.local", namespace)
	log.Printf("Using service discovery hostname: %s", targetname)

	for {
		ips, err := net.LookupIP(targetname)
		if err != nil {
			log.Fatalf("Cannot look up %s: %v", targetname, err)
		}
		sort.Slice(ips, func(i int, j int) bool {
			return bytes.Compare(ips[i], ips[j]) < 0
		})
		log.Printf("Found IPs: %v", ips)
		time.Sleep(5 * time.Second)
	}
}
