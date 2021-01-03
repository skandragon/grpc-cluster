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

func dumpEnv() {
	env := os.Environ()
	for _, item := range env {
		log.Printf("Environment: %s", item)
	}
}

func makeConfig() (*appConfig, error) {
	namespace := os.Getenv("POD_NAMESPACE")
	if len(namespace) == 0 {
		return nil, fmt.Errorf("environment variable POD_NAMESPACE is not set")
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

	return &appConfig{
		targetname: targetname,
		namespace:  namespace,
	}, nil
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
	dumpEnv()

	config, err := makeConfig()
	if err != nil {
		log.Panicf("Could not make config: %v", err)
	}

	log.Printf("Using service discovery hostname: %s", config.targetname)

	for {
		ips := getPeerAddresses()
		log.Printf("Found IPs: %v", ips)
		time.Sleep(5 * time.Second)
	}
}
