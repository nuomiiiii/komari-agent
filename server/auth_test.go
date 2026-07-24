package server

import (
	"net/http"
	"net/url"
	"testing"
)

func TestAgentAuthorizationUsesHeader(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://example.com/api/clients/v2/rpc", nil)
	if err != nil {
		t.Fatal(err)
	}
	authorizeAgentRequest(request, "secret-token")
	if got := request.Header.Get("Authorization"); got != "Bearer secret-token" {
		t.Fatalf("unexpected Authorization header: %q", got)
	}
	if request.URL.RawQuery != "" {
		t.Fatalf("agent credential leaked into query: %q", request.URL.RawQuery)
	}
}

func TestWebSocketEndpointDoesNotContainAgentToken(t *testing.T) {
	originalEndpoint, originalToken := flags.Endpoint, flags.Token
	flags.Endpoint, flags.Token = "https://example.com", "secret-token"
	t.Cleanup(func() {
		flags.Endpoint, flags.Token = originalEndpoint, originalToken
	})

	for _, protocolVersion := range []int{1, 2} {
		endpoint := buildWebSocketEndpoint(protocolVersion)
		parsed, err := url.Parse(endpoint)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.RawQuery != "" {
			t.Fatalf("v%d websocket URL contains query data: %q", protocolVersion, parsed.RawQuery)
		}
		if endpoint == "" || parsed.Host != "example.com" {
			t.Fatalf("unexpected websocket endpoint: %q", endpoint)
		}
	}
}
