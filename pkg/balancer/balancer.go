package balancer

import (
	"net/url"
	"sync"
)

type Balancer interface {
	Next() *Server
	AddServer(url *url.URL, varName string)
	GetServers() []*Server
	CountServers() int
}

type Server struct {
	URL        *url.URL
	IsDead     bool
	Wt         uint32
	HealthPath string
	Mu         sync.RWMutex
}

func (s *Server) SetDead(d bool) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.IsDead = d
}

func (s *Server) GetIsDead() bool {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	return s.IsDead
}

func (s *Server) GetURL() *url.URL {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	return s.URL
}
