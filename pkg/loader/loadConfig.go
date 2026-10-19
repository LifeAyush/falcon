package loader

import (
	"log"
	"net/url"
	"time"

	"github.com/LifeAyush/falcon/pkg/balancer"
	"github.com/LifeAyush/falcon/pkg/balancer/random"
	"github.com/LifeAyush/falcon/pkg/balancer/roundrobin"
	"github.com/LifeAyush/falcon/pkg/balancer/wroundrobin"
	"github.com/LifeAyush/falcon/pkg/constraints"
	"github.com/LifeAyush/falcon/pkg/constraints/status"
	"github.com/LifeAyush/falcon/pkg/dynamic"
	"github.com/LifeAyush/falcon/pkg/health"
	"github.com/LifeAyush/falcon/pkg/parsing"
	"github.com/LifeAyush/falcon/pkg/utils"
)

type BalancerType int

const (
	RoundRobinBalancer  BalancerType = 0
	RandomBalancer      BalancerType = 1
	WRoundRobinBalancer BalancerType = 2
)

type MatchType int

const (
	ExactMatch  MatchType = 0
	PrefixMatch MatchType = 1
	RegexMatch  MatchType = 2
)

type RouteInfo struct {
	Endpoint      string
	Match         MatchType
	TargetServer  *balancer.Server
	IsMultiTarget bool
	Balancer      balancer.Balancer
}

var (
	LimitReqCache *utils.ECache
	Routes        []*RouteInfo
)

func loadConfig() error {
	config := parsing.GetConfig()
	_ = status.Init()

	if config.LimitReq.Enable {
		LimitReqCache = utils.NewECache(time.Millisecond * time.Duration(config.LimitReq.Interval))
		constraints.InitLimitCache(LimitReqCache)
	}

	Routes = nil
	proxyConfig := config.Proxy
	var allServers []*balancer.Server
	for i := range proxyConfig.Routes {
		routeInfo := getRouteInfo(&proxyConfig.Routes[i])
		Routes = append(Routes, routeInfo)
		if routeInfo.IsMultiTarget {
			allServers = append(allServers, routeInfo.Balancer.GetServers()...)
		} else if routeInfo.TargetServer != nil {
			allServers = append(allServers, routeInfo.TargetServer)
		}
	}

	interval := time.Duration(config.Core.HealthIntervalMs) * time.Millisecond
	timeout := time.Duration(config.Core.HealthTimeoutMs) * time.Millisecond
	health.StartProbes(allServers, interval, timeout)

	return nil
}

// NewBalancerForRoute selects a balancer implementation (exported for tests).
func NewBalancerForRoute(bt BalancerType, urls []*url.URL, varNames []string, weights []uint32) balancer.Balancer {
	switch bt {
	case RandomBalancer:
		return random.NewBalancer(urls, varNames)
	case WRoundRobinBalancer:
		return wroundrobin.NewBalancer(urls, varNames, weights)
	default:
		return roundrobin.NewBalancer(urls, varNames)
	}
}

func getRouteInfo(routeConfig *parsing.Route) *RouteInfo {
	routeInfo := new(RouteInfo)

	if len(routeConfig.Backends) == 0 {
		panic(routeConfig.Endpoint + "Empty target")
	}

	routeInfo.Endpoint = routeConfig.Endpoint
	routeInfo.Match = MatchType(routeConfig.Match)

	if len(routeConfig.Backends) == 1 {
		routeInfo.IsMultiTarget = false
		u, err := url.Parse(routeConfig.Backends[0].Url)
		if err != nil {
			log.Fatal(err)
		}
		routeInfo.TargetServer = &balancer.Server{
			URL:        u,
			HealthPath: routeConfig.Backends[0].HealthPath,
		}
		if routeConfig.Backends[0].VarName != "" {
			dynamic.DyServers[routeConfig.Backends[0].VarName] = routeInfo.TargetServer
		}
		return routeInfo
	}

	routeInfo.IsMultiTarget = true
	urls := make([]*url.URL, 0, len(routeConfig.Backends))
	varNames := make([]string, 0, len(routeConfig.Backends))
	weights := make([]uint32, 0, len(routeConfig.Backends))
	healthPaths := make([]string, 0, len(routeConfig.Backends))
	for _, b := range routeConfig.Backends {
		curURL, err := url.Parse(b.Url)
		if err != nil {
			log.Fatal(err)
		}
		urls = append(urls, curURL)
		varNames = append(varNames, b.VarName)
		wt := b.Weight
		if wt < 1 {
			wt = 1
		}
		weights = append(weights, wt)
		healthPaths = append(healthPaths, b.HealthPath)
	}

	routeInfo.Balancer = NewBalancerForRoute(BalancerType(routeConfig.Balancer), urls, varNames, weights)
	for i, s := range routeInfo.Balancer.GetServers() {
		if i < len(healthPaths) {
			s.HealthPath = healthPaths[i]
		}
	}

	return routeInfo
}

func (routeInfo *RouteInfo) GetTargetServer() *balancer.Server {
	if !routeInfo.IsMultiTarget {
		if routeInfo.TargetServer != nil && routeInfo.TargetServer.GetIsDead() {
			return nil
		}
		return routeInfo.TargetServer
	}
	return routeInfo.Balancer.Next()
}

// Load reads the configured TOML file and builds routes. Call once at process start.
func Load() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Loaded %d routes", len(Routes))
}
