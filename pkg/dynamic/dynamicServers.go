package dynamic

import (
	"github.com/LifeAyush/falcon/pkg/balancer"
)

var (
	DyServers map[string]*balancer.Server
)

func init() {
	DyServers = make(map[string]*balancer.Server, 0)
}
