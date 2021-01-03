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
	config, err := makeConfig()
	if err != nil {
		t.Error(err)
	}
	if config.targetname != "xservice.xnamespace.svc.xsuffix.foo" {
		t.Errorf("Incorrect hostname generated: %s", config.targetname)
	}
}
