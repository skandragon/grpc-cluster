package main

import (
	"os"
	"testing"
)

func TestMakeConfig1(t *testing.T) {

	os.Setenv("POD_NAMESPACE", "xnamespace")
	config, err := makeConfig()
	if err != nil {
		t.Error(err)
	}
	if config.namespace != "xnamespace" {
		t.Errorf("expected namespace xnamespace, got %s", config.namespace)
	}
	if config.targetname != "grpc-cluster.xnamespace.svc.cluster.local" {
		t.Errorf("Incorrect hostname generated: %s", config.targetname)
	}
}

func TestMakeConfig2(t *testing.T) {

	os.Setenv("POD_NAMESPACE", "xnamespace")
	os.Setenv("SERVICE_NAME", "xservice")
	os.Setenv("DNS_SUFFIX", "xsuffix.foo")
	os.Setenv("POD_IPS", "1.2.3.4,5.6.7.8,fe80::1")
	config, err := makeConfig()
	if err != nil {
		t.Error(err)
	}
	if config.targetname != "xservice.xnamespace.svc.xsuffix.foo" {
		t.Errorf("Incorrect hostname generated: %s", config.targetname)
	}
	if config.myAddresses[0].String() != "1.2.3.4" {
		t.Error("Incorect addresses parsed (expected 1.2.3.4")
	}
	if config.myAddresses[1].String() != "5.6.7.8" {
		t.Error("Incorect addresses parsed (expected 5.6.7.8)")
	}
	if config.myAddresses[2].String() != "fe80::1" {
		t.Error("Incorect addresses parsed (expected fe80::1)")
	}
}
