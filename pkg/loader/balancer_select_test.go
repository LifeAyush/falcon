package loader

import (
	"net/url"
	"testing"
)

func TestNewBalancerForRouteTypes(t *testing.T) {
	urls := []*url.URL{
		mustURL(t, "http://a:1"),
		mustURL(t, "http://b:2"),
	}
	names := []string{"", ""}

	b0 := NewBalancerForRoute(RoundRobinBalancer, urls, names, nil)
	b1 := NewBalancerForRoute(RandomBalancer, urls, names, nil)
	b2 := NewBalancerForRoute(WRoundRobinBalancer, urls, names, []uint32{1, 2})

	if b0.CountServers() != 2 || b1.CountServers() != 2 || b2.CountServers() != 2 {
		t.Fatal("unexpected server counts")
	}

	servers := b2.GetServers()
	if servers[0].Wt != 1 || servers[1].Wt != 2 {
		t.Fatalf("WRR weights not applied: %d %d", servers[0].Wt, servers[1].Wt)
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
