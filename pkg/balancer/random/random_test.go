package random

import (
	"math/rand"
	"net/url"
	"testing"
)

func TestRandomDistribution(t *testing.T) {
	urls := []*url.URL{
		mustParse(t, "http://a:1"),
		mustParse(t, "http://b:2"),
		mustParse(t, "http://c:3"),
	}
	rr := NewBalancerWithRand(urls, nil, rand.New(rand.NewSource(42)))

	counts := map[string]int{}
	const n = 3000
	for i := 0; i < n; i++ {
		counts[rr.Next().URL.Host]++
	}
	for _, host := range []string{"a:1", "b:2", "c:3"} {
		if counts[host] < 500 {
			t.Fatalf("host %s under-represented: %d/%d (%v)", host, counts[host], n, counts)
		}
	}
}

func TestRandomSkipsDead(t *testing.T) {
	urls := []*url.URL{
		mustParse(t, "http://a:1"),
		mustParse(t, "http://b:2"),
	}
	rr := NewBalancerWithRand(urls, nil, rand.New(rand.NewSource(1)))
	rr.Servers[0].SetDead(true)
	for i := 0; i < 20; i++ {
		if rr.Next().URL.Host != "b:2" {
			t.Fatal("expected only live backend")
		}
	}
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
