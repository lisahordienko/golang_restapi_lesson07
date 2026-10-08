package main

import "testing"

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name       string
		certFile   string
		keyFile    string
		wantScheme string
		wantTLS    bool
	}{
		{
			name:       "defaults to HTTP without TLS files",
			wantScheme: "http",
			wantTLS:    false,
		},
		{
			name:       "uses HTTPS when TLS files are configured",
			certFile:   "certs/server.crt",
			keyFile:    "certs/server.key",
			wantScheme: "https",
			wantTLS:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := loadServerConfig(tt.certFile, tt.keyFile)
			if err != nil {
				t.Fatalf("loadServerConfig() error = %v", err)
			}
			if config.scheme != tt.wantScheme {
				t.Fatalf("schema = %q, want %q", config.scheme, tt.wantScheme)
			}
			if config.tls != tt.wantTLS {
				t.Fatalf("tls = %v, want %v", config.tls, tt.wantTLS)
			}
		})
	}
}
