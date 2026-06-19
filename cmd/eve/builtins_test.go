package main

import "testing"

func TestValidatePublicURLRejectsLocalAndBadScheme(t *testing.T) {
	// IP literals don't require DNS, so these are offline-safe.
	bad := []string{
		"http://127.0.0.1/",       // loopback
		"http://169.254.169.254/", // link-local (cloud metadata)
		"http://[::1]/",           // loopback v6
		"http://10.0.0.1/",        // private
		"http://0.0.0.0/",         // unspecified
		"ftp://example.com/",      // bad scheme
		"file:///etc/passwd",      // bad scheme
		"http:///nohost",          // no host
		"not a url",               // unparseable / no scheme
	}
	for _, raw := range bad {
		if err := validatePublicURL(raw); err == nil {
			t.Errorf("validatePublicURL(%q) = nil, want error", raw)
		}
	}
}
