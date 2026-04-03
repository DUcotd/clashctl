package mihomo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeProxyType(t *testing.T) {
	tests := map[string]string{
		"Selector":     "select",
		"URLTest":      "url-test",
		"Fallback":     "fallback",
		"load-balance": "load-balance",
		"Compatible":   "compatible",
	}

	for in, want := range tests {
		if got := NormalizeProxyType(in); got != want {
			t.Fatalf("NormalizeProxyType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsProxyGroupType(t *testing.T) {
	if !IsProxyGroupType("Selector") || !IsProxyGroupType("URLTest") {
		t.Fatal("expected Selector and URLTest to be treated as groups")
	}
	if IsProxyGroupType("Vless") || IsProxyGroupType("Trojan") {
		t.Fatal("expected node protocols not to be treated as groups")
	}
}

func TestGetAllProxyGroupsSendsAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("missing auth"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"proxies":{"PROXY":{"name":"PROXY","type":"Selector","all":["A"],"now":"A"}}}`))
	}))
	defer server.Close()

	client := NewClientWithSecret(server.URL, "secret-token")
	groups, err := client.GetAllProxyGroups()
	if err != nil {
		t.Fatalf("GetAllProxyGroups() error = %v", err)
	}
	if _, ok := groups["PROXY"]; !ok {
		t.Fatalf("GetAllProxyGroups() = %#v, want PROXY group", groups)
	}
}

func TestGetAllProxiesSendsAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("missing auth"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"proxies":{"NodeA":{"name":"NodeA","type":"Vless","provider-name":"airport"}}}`))
	}))
	defer server.Close()

	client := NewClientWithSecret(server.URL, "secret-token")
	proxies, err := client.GetAllProxies()
	if err != nil {
		t.Fatalf("GetAllProxies() error = %v", err)
	}
	if _, ok := proxies["NodeA"]; !ok {
		t.Fatalf("GetAllProxies() = %#v, want NodeA", proxies)
	}
}

func TestGetAllProxyGroupsRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"proxies":{"PROXY":{"name":"PROXY","type":"Selector","all":["` + strings.Repeat("A", APIResponseMaxSize) + `"],"now":"A"}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetAllProxyGroups()
	if err == nil {
		t.Fatal("GetAllProxyGroups() should reject oversized response bodies")
	}
	if !strings.Contains(err.Error(), "API 响应体过大") {
		t.Fatalf("GetAllProxyGroups() error = %v", err)
	}
}
