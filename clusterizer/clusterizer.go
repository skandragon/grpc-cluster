package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/skandragon/grpc-cluster/syncc"

	"google.golang.org/grpc"
)

var (
	prometheusPort = flag.Int("prometheusPort", 9102, "The HTTP port to serve /metrics for Prometheus")

	config *appConfig

	// metrics
	knownPeersGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "peers_known",
		Help: "The currently known peers",
	}, []string{})
	connectedPeersGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "peers_connected",
		Help: "The currently connected peers",
	}, []string{})
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
		donechan:  make(chan bool, 1),
		firstPing: 0,
		pingCount: 0,
	}
}

func isHostRemoved(donechan chan bool) bool {
	select {
	case _ = <-donechan:
		return true
	default:
		return false
	}
}

func pingLoop(donechan chan bool, target string) bool {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(10 * time.Second),
	}

	log.Printf("Connecting: %s", target)
	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		log.Printf("Could not connect: %s: %v", target, err)
		return false
	}
	defer conn.Close()

	client := syncc.NewSyncServiceClient(conn)
	if client == nil {
		log.Printf("Unable to connect: %s", target)
		return false
	}
	log.Printf("Connected: %s", target)
	connectedPeersGauge.WithLabelValues().Inc()

	for {
		if isHostRemoved(donechan) {
			connectedPeersGauge.WithLabelValues().Dec()
			return true
		}

		now := uint64(time.Now().UnixNano())
		//log.Printf("Sending ping: %s: %d", target, now)
		resp, err := client.Ping(context.Background(), &syncc.PingRequest{
			Ts: now,
		})
		if err != nil {
			log.Printf("Got error sending ping: %s: %v", target, err)
			connectedPeersGauge.WithLabelValues().Dec()
			return false
		}
		log.Printf("Got response from ping: %s: %d", target, resp.EchoedTs)

		time.Sleep(10 * time.Second)
	}
}

func connectAndPing(donechan chan bool, address net.IP) {
	target := fmt.Sprintf("%s:%d", address.String(), 9010)

	for {
		done := pingLoop(donechan, target)
		if done {
			log.Printf("Stopping connection to %s", target)
			return
		}

		if isHostRemoved(donechan) {
			log.Printf("Stopping connection to %s", target)
			return
		}
	}
}

func (s *syncServer) Ping(ctx context.Context, in *syncc.PingRequest) (*syncc.PingResponse, error) {
	return &syncc.PingResponse{
		Ts:       uint64(time.Now().UnixNano()),
		EchoedTs: in.Ts,
	}, nil
}

type syncServer struct {
	syncc.UnimplementedSyncServiceServer
}

func newServer() *syncServer {
	return &syncServer{}
}

func runGRPCServer() {
	lis, err := net.Listen("tcp", ":9010")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	syncc.RegisterSyncServiceServer(grpcServer, newServer())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start GRPC server: %v", err)
	}
}

func runHostTracking() {
	hosts := make(map[string]*hostInfo)
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
				hostinfo.donechan <- true
				log.Printf("Host %s removed", k)
				delete(hosts, k)
			}
		}
		log.Printf("Current host count: %d", len(hosts))
		knownPeersGauge.WithLabelValues().Set(float64(len(hosts)))
		time.Sleep(10 * time.Second)
	}
}

func runPrometheusHTTPServer(port int) {
	log.Printf("Running HTTP listener for Prometheus on port %d", port)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	server.ListenAndServe()

	prometheus.MustRegister(knownPeersGauge)
	prometheus.MustRegister(connectedPeersGauge)
}

func main() {
	dumpEnv()

	flag.Parse()

	cfg, err := makeConfig()
	if err != nil {
		log.Panicf("Could not make config: %v", err)
	}
	config = cfg
	log.Printf("Using service discovery hostname: %s", config.targetname)

	//
	// Run Prometheus HTTP server
	//
	if prometheusPort != nil {
		go runPrometheusHTTPServer(*prometheusPort)
	}

	go runGRPCServer()

	runHostTracking() // should never return
}
