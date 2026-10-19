package random

import (
	"math/rand"
	"net/url"

	"github.com/LifeAyush/falcon/pkg/balancer"
	"github.com/LifeAyush/falcon/pkg/dynamic"
)

type random struct {
	Servers []*balancer.Server
	rng     *rand.Rand
}

func NewBalancer(urls []*url.URL, varNames []string) *random {
	return NewBalancerWithRand(urls, varNames, rand.New(rand.NewSource(rand.Int63())))
}

func NewBalancerWithRand(urls []*url.URL, varNames []string, rng *rand.Rand) *random {
	rr := &random{rng: rng}
	for i := 0; i < len(urls); i++ {
		name := ""
		if i < len(varNames) {
			name = varNames[i]
		}
		rr.AddServer(urls[i], name)
	}
	return rr
}

func (rr *random) Next() *balancer.Server {
	n := len(rr.Servers)
	if n == 0 {
		return nil
	}
	alive := make([]*balancer.Server, 0, n)
	for _, s := range rr.Servers {
		if !s.GetIsDead() {
			alive = append(alive, s)
		}
	}
	if len(alive) == 0 {
		return nil
	}
	return alive[rr.rng.Intn(len(alive))]
}

func (rr *random) AddServer(u *url.URL, varName string) {
	srv := &balancer.Server{URL: u}
	if varName != "" {
		dynamic.DyServers[varName] = srv
	}
	rr.Servers = append(rr.Servers, srv)
}

func (rr *random) GetServers() []*balancer.Server {
	return rr.Servers
}

func (rr *random) CountServers() int {
	return len(rr.Servers)
}
