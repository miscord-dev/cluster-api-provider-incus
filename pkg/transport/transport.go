package transport

import (
	"fmt"
	"net/http"
	"os"

	incusclient "github.com/lxc/incus/client"
)

type TokenTransport struct {
	wrapped *http.Transport

	file      string
	tokenType string
}

func NewTransport(t *http.Transport, file string, tokenType string) *TokenTransport {
	return &TokenTransport{
		wrapped:   t,
		file:      file,
		tokenType: tokenType,
	}
}

var _ http.RoundTripper = &TokenTransport{}
var _ incusclient.HTTPTransporter = &TokenTransport{}

func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Load file
	token, err := os.ReadFile(t.file)
	if err != nil {
		return nil, fmt.Errorf("failed to load token file %s: %w", t.file, err)
	}

	// Set token
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", t.tokenType, token))

	// Perform request
	return t.wrapped.RoundTrip(req)
}

// Transport what this struct wraps
func (t *TokenTransport) Transport() *http.Transport {
	return t.wrapped
}
