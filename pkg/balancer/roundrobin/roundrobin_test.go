package roundrobin

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

func TestRoundRobinSequence(t *testing.T) {
	urls := []*url.URL{
		mustURL(t, "http://a:1"),
		mustURL(t, "http://b:2"),
		mustURL(t, "http://c:3"),
	}
	rr := NewBalancer(urls, []string{"", "", ""})

	got := make([]string, 6)
	for i := 0; i < 6; i++ {
		got[i] = rr.Next().URL.Host
	}
	want := []string{"a:1", "b:2", "c:3", "a:1", "b:2", "c:3"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("step %d: got %s want %s (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestRoundRobinSkipsDead(t *testing.T) {
	urls := []*url.URL{
		mustURL(t, "http://a:1"),
		mustURL(t, "http://b:2"),
	}
	rr := NewBalancer(urls, nil)
	rr.Servers[0].SetDead(true)

	for i := 0; i < 4; i++ {
		s := rr.Next()
		if s == nil || s.URL.Host != "b:2" {
			t.Fatalf("expected only b:2, got %#v", s)
		}
	}

	rr.Servers[1].SetDead(true)
	if rr.Next() != nil {
		t.Fatal("expected nil when all dead")
	}
}
