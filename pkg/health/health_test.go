package health

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProbeGET(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer badSrv.Close()

	okURL, _ := url.Parse(okSrv.URL)
	badURL, _ := url.Parse(badSrv.URL)
	client := okSrv.Client()

	if !ProbeGET(client, okURL, "/health") {
		t.Fatal("expected healthy")
	}
	if ProbeGET(client, badURL, "/health") {
		t.Fatal("expected unhealthy")
	}
	if !ProbeGET(client, okURL, "") {
		t.Fatal("empty path should skip and return healthy")
	}
}
