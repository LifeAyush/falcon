package wroundrobin

import (
	"net/url"
	"testing"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestWeightedRatios(t *testing.T) {
	urls := []*url.URL{
		mustURL(t, "http://a:1"),
		mustURL(t, "http://b:2"),
		mustURL(t, "http://c:3"),
	}
	wrr := NewBalancer(urls, nil, []uint32{1, 2, 3})

	counts := map[string]int{}
	const n = 600
	for i := 0; i < n; i++ {
		counts[wrr.Next().URL.Host]++
	}
	// Exact over full cycles of total weight 6: 100, 200, 300
	if counts["a:1"] != 100 || counts["b:2"] != 200 || counts["c:3"] != 300 {
		t.Fatalf("unexpected weights over %d draws: %v", n, counts)
	}
}

func TestWeightedSkipsDead(t *testing.T) {
	urls := []*url.URL{
		mustURL(t, "http://a:1"),
		mustURL(t, "http://b:2"),
	}
	wrr := NewBalancer(urls, nil, []uint32{1, 1})
	wrr.Servers[0].SetDead(true)
	for i := 0; i < 10; i++ {
		s := wrr.Next()
		if s == nil || s.URL.Host != "b:2" {
			t.Fatalf("got %#v", s)
		}
	}
}

func TestWeightedSingleServer(t *testing.T) {
	urls := []*url.URL{mustURL(t, "http://a:1")}
	wrr := NewBalancer(urls, nil, []uint32{5})
	if wrr.Next().URL.Host != "a:1" {
		t.Fatal("expected single backend")
	}
}
