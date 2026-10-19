package roundrobin

import (
	"net/url"
	"sync/atomic"

	"github.com/LifeAyush/falcon/pkg/balancer"
	"github.com/LifeAyush/falcon/pkg/dynamic"
)

type roundrobin struct {
	Servers []*balancer.Server
	next    uint32
}

func NewBalancer(urls []*url.URL, varNames []string) *roundrobin {
	rr := &roundrobin{}
	for i := 0; i < len(urls); i++ {
		name := ""
		if i < len(varNames) {
			name = varNames[i]
		}
		rr.AddServer(urls[i], name)
	}
	return rr
}

func (rr *roundrobin) Next() *balancer.Server {
	n := len(rr.Servers)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		idx := atomic.AddUint32(&rr.next, 1)
		s := rr.Servers[int(idx-1)%n]
		if !s.GetIsDead() {
			return s
		}
	}
	return nil
}

func (rr *roundrobin) AddServer(u *url.URL, varName string) {
	srv := &balancer.Server{URL: u}
	if varName != "" {
		dynamic.DyServers[varName] = srv
	}
	rr.Servers = append(rr.Servers, srv)
}

func (rr *roundrobin) GetServers() []*balancer.Server {
	return rr.Servers
}

func (rr *roundrobin) CountServers() int {
	return len(rr.Servers)
}
