// Copyright 2025 The Cloud Native Events Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package restapi

import (
	"crypto/tls"
	"net"
	"testing"
)

func TestIsBlockedDialIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		// Allowed: loopback (RHT-0003 in-pod) and private (cluster Pod IPs).
		{"127.0.0.1", false},
		{"::1", false},
		{"10.0.0.5", false},
		{"172.16.3.4", false},
		{"192.168.1.10", false},
		{"fd12:3456::1", false},
		{"93.184.216.34", false}, // public
		// Blocked: link-local, cloud metadata, multicast, unspecified.
		{"169.254.0.1", true},
		{"169.254.169.254", true},
		{"fe80::1", true},
		{"fd00:ec2::254", true},
		{"224.0.0.1", true},
		{"0.0.0.0", true},
		{"::", true},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("bad test IP %q", c.ip)
		}
		if got := isBlockedDialIP(ip); got != c.blocked {
			t.Errorf("isBlockedDialIP(%s) = %v, want %v", c.ip, got, c.blocked)
		}
	}
	if !isBlockedDialIP(nil) {
		t.Errorf("isBlockedDialIP(nil) should be blocked")
	}
}

func TestValidateEndpointURI(t *testing.T) {
	valid := []string{
		"http://localhost:8080/event",
		"https://consumer.ptp.svc.cluster.local:9043/ack",
		"http://10.128.0.9:8080/callback",
	}
	for _, u := range valid {
		if err := validateEndpointURI(u); err != nil {
			t.Errorf("validateEndpointURI(%q) = %v, want nil", u, err)
		}
	}
	invalid := []string{
		"ftp://host/x",                  // bad scheme
		"http://",                       // empty host
		"://nope",                       // unparseable scheme
		"http://169.254.169.254/latest", // metadata
		"http://[fe80::1]/x",            // link-local
		"http://224.0.0.1/x",            // multicast
		"http://0.0.0.0/x",              // unspecified
	}
	for _, u := range invalid {
		if err := validateEndpointURI(u); err == nil {
			t.Errorf("validateEndpointURI(%q) = nil, want error", u)
		}
	}
}

func TestIsLoopbackRemoteAddr(t *testing.T) {
	yes := []string{"127.0.0.1:5000", "[::1]:5000", "localhost:5000", "127.0.0.1"}
	no := []string{"", "10.0.0.1:5000", "192.168.1.1:80", "example.com:80"}
	for _, a := range yes {
		if !isLoopbackRemoteAddr(a) {
			t.Errorf("isLoopbackRemoteAddr(%q) = false, want true", a)
		}
	}
	for _, a := range no {
		if isLoopbackRemoteAddr(a) {
			t.Errorf("isLoopbackRemoteAddr(%q) = true, want false", a)
		}
	}
}

func TestApplyTLSProfile(t *testing.T) {
	// Explicit profile is applied verbatim.
	c := &AuthConfig{
		TLSMinVersion:   "VersionTLS13",
		TLSCipherSuites: []string{"TLS_AES_128_GCM_SHA256", "bogus-name"},
	}
	cfg := &tls.Config{}
	c.ApplyTLSProfile(cfg)
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %x, want TLS13", cfg.MinVersion)
	}

	// No profile => default floor of TLS 1.2, existing MinVersion preserved.
	empty := &AuthConfig{}
	cfg2 := &tls.Config{}
	empty.ApplyTLSProfile(cfg2)
	if cfg2.MinVersion != tls.VersionTLS12 {
		t.Errorf("default MinVersion = %x, want TLS12", cfg2.MinVersion)
	}

	// Nil receiver must not panic.
	var nilCfg *AuthConfig
	nilCfg.ApplyTLSProfile(&tls.Config{})
}
