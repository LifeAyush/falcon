package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"

	"github.com/LifeAyush/falcon/pkg/balancer"
	"github.com/LifeAyush/falcon/pkg/constraints"
	"github.com/LifeAyush/falcon/pkg/constraints/status"
	"github.com/LifeAyush/falcon/pkg/loader"
	"github.com/LifeAyush/falcon/pkg/parsing"
	"golang.org/x/net/netutil"
)

type Proxy struct{}

func (p *Proxy) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if parsing.GetConfig().LimitReq.Enable {
		ip := strings.Split(req.RemoteAddr, ":")[0]
		if constraints.ExceededLimitReq(ip, req) {
			http.Error(res, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
	}

	if parsing.GetConfig().Core.EnableServerStats {
		status.Instance().AddReqCount()
		defer status.Instance().SubReqCount()
	}

	route := new(loader.RouteInfo)
	existRoute := false

	for _, r := range loader.Routes {
		switch r.Match {
		case loader.ExactMatch:
			if req.URL.Path == r.Endpoint {
				route = r
				existRoute = true
			}
		case loader.PrefixMatch:
			if strings.HasPrefix(req.URL.Path, r.Endpoint) {
				route = r
				existRoute = true
			}
		case loader.RegexMatch:
			if match, _ := regexp.MatchString(r.Endpoint, req.URL.Path); match {
				route = r
				existRoute = true
			}
		}
	}

	if !existRoute {
		http.NotFound(res, req)
		return
	}

	target := route.GetTargetServer()
	if target == nil {
		http.Error(res, "Bad Gateway", http.StatusBadGateway)
		return
	}

	proxy := newHostReverseProxy(target)
	proxy.ServeHTTP(res, req)
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func newHostReverseProxy(server *balancer.Server) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		target := server.GetURL()
		targetQuery := target.RawQuery
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		req.Host = target.Host

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
		xForwarded := req.Header["X-Forwarded-For"]
		xForwarded = append(xForwarded, req.RemoteAddr)
		req.Header["X-Forwarded-For"] = xForwarded
	}
	return &httputil.ReverseProxy{Director: director}
}

type GateServer struct{}

func (s *GateServer) proxy80() *http.Server {
	port := parsing.GetConfig().Core.Listen
	if port == 0 {
		port = 80
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	if parsing.GetConfig().Core.LimitMaxConn > 0 {
		ln = netutil.LimitListener(ln, parsing.GetConfig().Core.LimitMaxConn)
	}

	p := &Proxy{}
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: p}
	if parsing.GetConfig().Core.ReadTimeout > 0 {
		srv.ReadTimeout = time.Duration(parsing.GetConfig().Core.ReadTimeout) * time.Second
	}
	if parsing.GetConfig().Core.WriteTimeout > 0 {
		srv.WriteTimeout = time.Duration(parsing.GetConfig().Core.WriteTimeout) * time.Second
	}
	if parsing.GetConfig().Core.IdleTimeout > 0 {
		srv.IdleTimeout = time.Duration(parsing.GetConfig().Core.IdleTimeout) * time.Second
	}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			panic(err.Error())
		}
	}()
	return srv
}

func (s *GateServer) Run() []*http.Server {
	loader.Load()
	return []*http.Server{s.proxy80()}
}
