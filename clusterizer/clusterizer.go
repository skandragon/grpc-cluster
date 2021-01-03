package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/skandragon/grpc-cluster/syncc"

	"google.golang.org/grpc"
)

var (
	config *appConfig
)

type appConfig struct {
	targetname  string
	namespace   string
	myAddresses []net.IP
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

	myAddresses := make([]net.IP, 0)
	myAddressStrings := strings.Split(os.Getenv("POD_IPS"), ",")
	for _, as := range myAddressStrings {
		addr := net.ParseIP(as)
		if myAddresses != nil {
			myAddresses = append(myAddresses, addr)
		}
	}

	return &appConfig{
		targetname:  targetname,
		namespace:   namespace,
		myAddresses: myAddresses,
	}, nil
}

func notMyAddress(address net.IP) bool {
	for _, ip := range config.myAddresses {
		if ip.String() == address.String() {
			return false
		}
	}
	return true
}

func getPeerAddresses() []net.IP {
	allIPs, err := net.LookupIP(config.targetname)
	if err != nil {
		log.Printf("Cannot look up %s: %v", config.targetname, err)
		return []net.IP{}
	}
	ips := make([]net.IP, 0)
	for _, ip := range allIPs {
		if notMyAddress(ip) {
			ips = append(ips, ip)
		}
	}
	sort.Slice(ips, func(i int, j int) bool {
		return bytes.Compare(ips[i], ips[j]) < 0
	})
	return ips
}

type hostInfo struct {
	address   net.IP
	donechan  chan bool
	firstPing uint64
	pingCount uint64
}

func makeHostInfo(address net.IP) *hostInfo {
	return &hostInfo{
		address:   address,
		donechan:  make(chan bool),
		firstPing: 0,
		pingCount: 0,
	}
}

func connectAndPing(donechan chan bool, address net.IP) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(10 * time.Second),
	}

	target := fmt.Sprintf("%s:%d", address.String(), 9010)

	for {
		conn, err := grpc.Dial(target, opts...)
		if err != nil {
			log.Printf("Could not connect: %v", err)
		}
		defer conn.Close()

		client := syncc.NewSyncServiceClient(conn)
		if client == nil {
			log.Printf("Unable to connect to %s", target)
		}

		for {
			if _, more := <-donechan; more == false {
				log.Printf("Stopping connection to %s", target)
				return
			}

			now := uint64(time.Now().UnixNano())

			log.Printf("Sending ping to %s", target)
			resp, err := client.Ping(context.Background(), &syncc.PingRequest{
				Ts: now,
			})
			if err != nil {
				log.Printf("Got error sending ping: %v", err)
				break
			}
			log.Printf("Got response from ping: %d", resp.EchoedTs)

			time.Sleep(10 * time.Second)
		}
		if _, more := <-donechan; more == false {
			log.Printf("Stopping connection to %s", target)
			return
		}
	}
}

func main() {
	hosts := make(map[string]*hostInfo)

	dumpEnv()

	cfg, err := makeConfig()
	if err != nil {
		log.Panicf("Could not make config: %v", err)
	}
	config = cfg
	log.Printf("Using service discovery hostname: %s", config.targetname)

	for {
		ips := getPeerAddresses()
		current := make(map[string]bool)
		for _, ip := range ips {
			current[ip.String()] = true
			if hosts[ip.String()] == nil {
				hosts[ip.String()] = makeHostInfo(ip)
				log.Printf("Detected new address: %s", ip.String())
				go connectAndPing(hosts[ip.String()].donechan, ip)
			}
		}
		for k, hostinfo := range hosts {
			if !current[k] {
				close(hostinfo.donechan)
				log.Printf("Host %s removed", k)
			}
		}
		time.Sleep(10 * time.Second)
	}
}
