package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/LifeAyush/falcon/pkg/balancer"
	"github.com/LifeAyush/falcon/pkg/loader"
	"github.com/LifeAyush/falcon/pkg/parsing"
)

// setup points the global config at an empty TOML file and installs a single
// prefix route for "/" that targets server.
func setup(tb testing.TB, server *balancer.Server) {
	tb.Helper()
	cfg := filepath.Join(tb.TempDir(), "falcon.toml")
	if err := os.WriteFile(cfg, []byte(""), 0o600); err != nil {
		tb.Fatal(err)
	}
	parsing.ResetForTest()
	parsing.ConfigFile.FilePath = cfg
	oldRoutes := loader.Routes
	loader.Routes = []*loader.RouteInfo{{
		Endpoint:     "/",
		Match:        loader.PrefixMatch,
		TargetServer: server,
	}}
	tb.Cleanup(func() {
		loader.Routes = oldRoutes
		parsing.ResetForTest()
	})
}

func mustParse(tb testing.TB, raw string) *url.URL {
	tb.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		tb.Fatal(err)
	}
	return u
}

func TestProxiesToBackendWithQueryAndXFF(t *testing.T) {
	var gotPath, gotQuery, gotXFF string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		gotXFF = strings.Join(r.Header["X-Forwarded-For"], ",")
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	// The target URL carries its own path and query, which must be joined
	// with the request's.
	setup(t, &balancer.Server{URL: mustParse(t, backend.URL+"/base?a=1")})

	req := httptest.NewRequest("GET", "/x/y?b=2", nil)
	req.RemoteAddr = "10.1.2.3:4567"
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	rec := httptest.NewRecorder()
	(&Proxy{}).ServeHTTP(rec, req)

	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("unexpected response: %d %q", rec.Code, rec.Body.String())
	}
	if gotPath != "/base/x/y" {
		t.Errorf("path = %q, want /base/x/y", gotPath)
	}
	if gotQuery != "a=1&b=2" {
		t.Errorf("query = %q, want a=1&b=2", gotQuery)
	}
	if !strings.HasPrefix(gotXFF, "9.9.9.9, 10.1.2.3:4567") {
		t.Errorf("X-Forwarded-For = %q, want existing value then client address", gotXFF)
	}
}

func TestRoutesToTheRightBackend(t *testing.T) {
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("a")) }))
	defer a.Close()
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("b")) }))
	defer b.Close()

	setup(t, nil)
	loader.Routes = []*loader.RouteInfo{
		{Endpoint: "/a", Match: loader.PrefixMatch, TargetServer: &balancer.Server{URL: mustParse(t, a.URL)}},
		{Endpoint: "/b", Match: loader.PrefixMatch, TargetServer: &balancer.Server{URL: mustParse(t, b.URL)}},
	}

	for path, want := range map[string]string{"/a/1": "a", "/b/1": "b"} {
		rec := httptest.NewRecorder()
		(&Proxy{}).ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Body.String() != want {
			t.Errorf("%s -> %q, want %q", path, rec.Body.String(), want)
		}
	}
}

func TestBackendURLChangeTakesEffectOnNextRequest(t *testing.T) {
	var hitsA, hitsB int32
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&hitsA, 1) }))
	defer a.Close()
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&hitsB, 1) }))
	defer b.Close()

	server := &balancer.Server{URL: mustParse(t, a.URL)}
	setup(t, server)

	serve := func() {
		(&Proxy{}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}
	serve()

	// Mirrors what the management API does for POST /apiStatus/backendChange.
	server.Mu.Lock()
	server.URL = mustParse(t, b.URL)
	server.Mu.Unlock()
	serve()

	if atomic.LoadInt32(&hitsA) != 1 || atomic.LoadInt32(&hitsB) != 1 {
		t.Fatalf("hits a=%d b=%d, want 1 and 1", hitsA, hitsB)
	}
}

func TestReverseProxyIsReusedPerBackend(t *testing.T) {
	s1 := &balancer.Server{URL: mustParse(t, "http://127.0.0.1:1")}
	s2 := &balancer.Server{URL: mustParse(t, "http://127.0.0.1:2")}
	if reverseProxyFor(s1) != reverseProxyFor(s1) {
		t.Error("same backend returned different ReverseProxy instances")
	}
	if reverseProxyFor(s1) == reverseProxyFor(s2) {
		t.Error("different backends share a ReverseProxy")
	}
}

func BenchmarkServeHTTP(b *testing.B) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer backend.Close()
	setup(b, &balancer.Server{URL: mustParse(b, backend.URL)})

	p := &Proxy{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/hello?x=1", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}
