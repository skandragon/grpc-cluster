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

var (
	config *appConfig
)

type appConfig struct {
	targetname string
	namespace  string
}

func makeConfig() *appConfig {
	env := os.Environ()
	for _, item := range env {
		log.Printf("Environment: %s", item)
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if len(namespace) == 0 {
		log.Fatal("Environment variable POD_NAMESPACE is not set.")
	}

	service := os.Getenv("SERVICE_NAME")
	if len(service) == 0 {
		service = "grpc-cluster"
	}

	dnsSuffix := os.Getenv("DNS_SUFFIX")
	if len(dnsSuffix) == 0 {
		dnsSuffix = "cluster.local"
	}

	targetname := fmt.Sprintf("%s.%s.svc.%s", service, namespace, dnsSuffix)
	log.Printf("Using service discovery hostname: %s", targetname)

	return &appConfig{
		targetname: targetname,
		namespace:  namespace,
	}
}

func getPeerAddresses() []net.IP {
	ips, err := net.LookupIP(config.targetname)
	if err != nil {
		log.Fatalf("Cannot look up %s: %v", config.targetname, err)
	}
	sort.Slice(ips, func(i int, j int) bool {
		return bytes.Compare(ips[i], ips[j]) < 0
	})
	return ips
}

func main() {
	config = makeConfig()

	for {
		ips := getPeerAddresses()
		log.Printf("Found IPs: %v", ips)
		time.Sleep(5 * time.Second)
	}
}
