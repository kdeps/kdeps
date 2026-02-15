package llm

import (
	"net/url"
	"testing"
)

func TestParseHostPortFromURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		defaultHost string
		defaultPort int
		wantHost    string
		wantPort    int
	}{
		{
			name:        "Empty URL returns defaults",
			baseURL:     "",
			defaultHost: "myhost",
			defaultPort: 8080,
			wantHost:    "myhost",
			wantPort:    8080,
		},
		{
			name:        "Empty URL with empty default host uses localhost",
			baseURL:     "",
			defaultHost: "",
			defaultPort: 11434,
			wantHost:    "localhost",
			wantPort:    11434,
		},
		{
			name:        "Valid HTTP URL with port",
			baseURL:     "http://example.com:9000",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "example.com",
			wantPort:    9000,
		},
		{
			name:        "Valid HTTPS URL without explicit port",
			baseURL:     "https://api.openai.com/v1",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "api.openai.com",
			wantPort:    8080, // Falls back to default when no port specified
		},
		{
			name:        "Localhost URL with port",
			baseURL:     "http://localhost:11434",
			defaultHost: "defaulthost",
			defaultPort: 8080,
			wantHost:    "localhost",
			wantPort:    11434,
		},
		{
			name:        "URL without scheme triggers manual parsing",
			baseURL:     "invalid-url-no-scheme",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "localhost",
			wantPort:    8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotPort := parseHostPortFromURL(tt.baseURL, tt.defaultHost, tt.defaultPort)
			if gotHost != tt.wantHost {
				t.Errorf("parseHostPortFromURL() gotHost = %v, want %v", gotHost, tt.wantHost)
			}
			if gotPort != tt.wantPort {
				t.Errorf("parseHostPortFromURL() gotPort = %v, want %v", gotPort, tt.wantPort)
			}
		})
	}
}

func TestExtractHostPortManually(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		defaultHost string
		defaultPort int
		wantHost    string
		wantPort    int
	}{
		{
			name:        "URL without scheme returns defaults",
			baseURL:     "no-scheme-here",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "localhost",
			wantPort:    8080,
		},
		{
			name:        "Valid scheme with host and port",
			baseURL:     "http://example.com:9000/path",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "example.com",
			wantPort:    9000,
		},
		{
			name:        "Valid scheme with host no port",
			baseURL:     "https://api.service.com/endpoint",
			defaultHost: "localhost",
			defaultPort: 443,
			wantHost:    "api.service.com",
			wantPort:    443,
		},
		{
			name:        "Scheme with invalid port uses default",
			baseURL:     "http://example.com:invalid/",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "example.com",
			wantPort:    8080,
		},
		{
			name:        "Empty scheme part returns defaults",
			baseURL:     "://example.com:9000",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "localhost",
			wantPort:    8080,
		},
		{
			name:        "Just scheme returns defaults",
			baseURL:     "http://",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "localhost",
			wantPort:    8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotPort := extractHostPortManually(tt.baseURL, tt.defaultHost, tt.defaultPort)
			if gotHost != tt.wantHost {
				t.Errorf("extractHostPortManually() gotHost = %v, want %v", gotHost, tt.wantHost)
			}
			if gotPort != tt.wantPort {
				t.Errorf("extractHostPortManually() gotPort = %v, want %v", gotPort, tt.wantPort)
			}
		})
	}
}

func TestExtractHostPortFromParsedURL(t *testing.T) {
	tests := []struct {
		name        string
		urlStr      string
		defaultHost string
		defaultPort int
		wantHost    string
		wantPort    int
	}{
		{
			name:        "URL with explicit port",
			urlStr:      "http://example.com:9000/path",
			defaultHost: "localhost",
			defaultPort: 8080,
			wantHost:    "example.com",
			wantPort:    9000,
		},
		{
			name:        "URL without port uses default",
			urlStr:      "https://api.example.com/v1",
			defaultHost: "localhost",
			defaultPort: 443,
			wantHost:    "api.example.com",
			wantPort:    443,
		},
		{
			name:        "URL with empty hostname uses default",
			urlStr:      "http:///path",
			defaultHost: "mydefault",
			defaultPort: 8080,
			wantHost:    "mydefault",
			wantPort:    8080,
		},
		{
			name:        "Localhost with custom port",
			urlStr:      "http://localhost:11434",
			defaultHost: "defaulthost",
			defaultPort: 8080,
			wantHost:    "localhost",
			wantPort:    11434,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse URL first
			parsedURL, err := url.Parse(tt.urlStr)
			if err != nil {
				t.Fatalf("Failed to parse URL %s: %v", tt.urlStr, err)
			}

			gotHost, gotPort := extractHostPortFromParsedURL(parsedURL, tt.defaultHost, tt.defaultPort)
			if gotHost != tt.wantHost {
				t.Errorf("extractHostPortFromParsedURL() gotHost = %v, want %v", gotHost, tt.wantHost)
			}
			if gotPort != tt.wantPort {
				t.Errorf("extractHostPortFromParsedURL() gotPort = %v, want %v", gotPort, tt.wantPort)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{
			name: "First is smaller",
			a:    5,
			b:    10,
			want: 5,
		},
		{
			name: "Second is smaller",
			a:    10,
			b:    5,
			want: 5,
		},
		{
			name: "Equal values",
			a:    7,
			b:    7,
			want: 7,
		},
		{
			name: "Negative values",
			a:    -5,
			b:    -10,
			want: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minInt(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("minInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
