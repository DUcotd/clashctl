package netsec

import (
	"net"
	"testing"
)

func TestIsPrivateIP_Public(t *testing.T) {
	publicIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"142.250.80.46",
		"203.0.113.1",
		"198.51.100.1",
	}
	for _, ipStr := range publicIPs {
		ip := net.ParseIP(ipStr)
		if IsPrivateIP(ip) {
			t.Errorf("IsPrivateIP(%s) = true, want false", ipStr)
		}
	}
}

func TestIsPrivateIP_Private(t *testing.T) {
	privateIPs := []string{
		"127.0.0.1",
		"127.0.0.255",
		"10.0.0.1",
		"10.255.255.255",
		"172.16.0.1",
		"172.31.255.255",
		"192.168.0.1",
		"192.168.255.255",
		"100.64.0.1",
		"100.127.255.255",
		"169.254.0.1",
		"169.254.255.255",
		"169.254.169.254",
		"::1",
		"fc00::1",
		"fd00::1",
		"fe80::1",
	}
	for _, ipStr := range privateIPs {
		ip := net.ParseIP(ipStr)
		if !IsPrivateIP(ip) {
			t.Errorf("IsPrivateIP(%s) = false, want true", ipStr)
		}
	}
}

func TestIsPrivateIP_EdgeCases(t *testing.T) {
	if IsPrivateIP(nil) {
		t.Error("IsPrivateIP(nil) = true, want false")
	}
	if IsPrivateIP(net.IP{}) {
		t.Error("IsPrivateIP(empty) = true, want false")
	}
}
