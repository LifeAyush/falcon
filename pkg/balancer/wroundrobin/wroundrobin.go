package wroundrobin

import (
	"net/url"
	"sync/atomic"

	"github.com/LifeAyush/falcon/pkg/balancer"
	"github.com/LifeAyush/falcon/pkg/dynamic"
)

type wroundrobin struct {
	Servers []*balancer.Server
	wts     []uint32
	totalWt uint32
	next    uint32
}

func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func (wrr *wroundrobin) reduce() {
	if len(wrr.Servers) == 0 || wrr.totalWt == 0 {
		return
	}
	g := wrr.Servers[0].Wt
	for _, s := range wrr.Servers[1:] {
		g = gcd(g, s.Wt)
	}
	if g <= 1 {
		return
	}
	wrr.totalWt /= g
	for i := range wrr.Servers {
		wrr.Servers[i].Wt /= g
	}
	wrr.wts = wrr.wts[:0]
	var cum uint32
	for _, s := range wrr.Servers {
		cum += s.Wt
		wrr.wts = append(wrr.wts, cum)
	}
}

func NewBalancer(urls []*url.URL, varNames []string, weights []uint32) *wroundrobin {
	wrr := &wroundrobin{}
	for i := 0; i < len(urls); i++ {
		name := ""
		if i < len(varNames) {
			name = varNames[i]
		}
		wt := uint32(1)
		if i < len(weights) && weights[i] > 0 {
			wt = weights[i]
		}
		wrr.addServer(urls[i], name, wt)
	}
	wrr.reduce()
	return wrr
}

func upperBound(wts []uint32, x uint32) int {
	n := len(wts)
	l, r := 0, n-1
	for l < r {
		mid := (l + r) / 2
		if wts[mid] < x {
			l = mid + 1
		} else {
			r = mid
		}
	}
	return l
}

func (wrr *wroundrobin) Next() *balancer.Server {
	n := len(wrr.Servers)
	if n == 0 || wrr.totalWt == 0 {
		return nil
	}
	for try := 0; try < n; try++ {
		seq := atomic.AddUint32(&wrr.next, 1)
		slot := (seq-1)%wrr.totalWt + 1
		ind := upperBound(wrr.wts, slot)
		s := wrr.Servers[ind]
		if !s.GetIsDead() {
			return s
		}
	}
	return nil
}

func (wrr *wroundrobin) addServer(u *url.URL, varName string, weight uint32) {
	if weight < 1 {
		weight = 1
	}
	srv := &balancer.Server{URL: u, Wt: weight}
	if varName != "" {
		dynamic.DyServers[varName] = srv
	}
	wrr.Servers = append(wrr.Servers, srv)
	wrr.totalWt += weight
	prev := uint32(0)
	if len(wrr.wts) > 0 {
		prev = wrr.wts[len(wrr.wts)-1]
	}
	wrr.wts = append(wrr.wts, prev+weight)
}

func (wrr *wroundrobin) AddServer(u *url.URL, varName string) {
	wrr.addServer(u, varName, 1)
	wrr.reduce()
}

func (wrr *wroundrobin) GetServers() []*balancer.Server {
	return wrr.Servers
}

func (wrr *wroundrobin) CountServers() int {
	return len(wrr.Servers)
}
