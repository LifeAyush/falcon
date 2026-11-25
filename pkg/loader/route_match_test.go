package loader

import (
	"strings"
	"testing"

	"github.com/LifeAyush/falcon/pkg/parsing"
)

func newRoute(t *testing.T, endpoint string, match int) *RouteInfo {
	t.Helper()
	r, err := getRouteInfo(&parsing.Route{
		Endpoint: endpoint,
		Match:    match,
		Backends: []parsing.Backend{{Url: "http://127.0.0.1:1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRouteMatching(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		match    int
		path     string
		want     bool
	}{
		{"exact hit", "/api", 0, "/api", true},
		{"exact miss on longer path", "/api", 0, "/api/x", false},
		{"prefix hit", "/api", 1, "/api/x", true},
		{"prefix miss", "/api", 1, "/other", false},
		{"regex hit", `^/user/[0-9]+$`, 2, "/user/42", true},
		{"regex miss", `^/user/[0-9]+$`, 2, "/user/abc", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := newRoute(t, c.endpoint, c.match).Matches(c.path); got != c.want {
				t.Fatalf("Matches(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

func TestRegexCompiledOnceAtLoad(t *testing.T) {
	r := newRoute(t, `^/a/.*`, 2)
	if r.Regex == nil {
		t.Fatal("regex route should carry a compiled regexp")
	}
	if newRoute(t, "/a", 0).Regex != nil {
		t.Fatal("non-regex route should not compile a regexp")
	}
}

func TestInvalidRegexRejectedAtLoad(t *testing.T) {
	_, err := getRouteInfo(&parsing.Route{
		Endpoint: "([unclosed",
		Match:    2,
		Backends: []parsing.Backend{{Url: "http://127.0.0.1:1"}},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("expected invalid regex error, got %v", err)
	}
}

// First matching route in config order wins (as ServeHTTP does).
func TestFirstMatchWinsOrdering(t *testing.T) {
	routes := []*RouteInfo{
		newRoute(t, "/api", 1),
		newRoute(t, `^/api/v1`, 2),
	}
	first := func(path string) *RouteInfo {
		for _, r := range routes {
			if r.Matches(path) {
				return r
			}
		}
		return nil
	}
	if first("/api/v1/x") != routes[0] {
		t.Fatal("earlier prefix route should win over later regex route")
	}
	if first("/nope") != nil {
		t.Fatal("no route should match")
	}
}
