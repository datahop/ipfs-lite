package config

import (
	"strings"
	"testing"
)

func TestConfigInit(t *testing.T) {
	conf, err := NewConfig("0")
	if err != nil {
		t.Fatal(err)
	}
	if conf.Identity.PeerID == "" || conf.Identity.PrivKey == "" {
		t.Fatal("Could not create identity properly")
	}
	if !strings.HasSuffix(conf.Addresses.Swarm[0], SwarmPort) {
		t.Fatal("Wrong swarm port")
	}
}

func TestConfigInitCustomPort(t *testing.T) {
	conf, err := NewConfig("5000")
	if err != nil {
		t.Fatal(err)
	}

	if strings.HasSuffix(conf.Addresses.Swarm[0], SwarmPort) {
		t.Fatal("Wrong swarm port")
	}
	if !strings.HasSuffix(conf.Addresses.Swarm[0], "5000") {
		t.Fatal("Wrong swarm port")
	}
}
