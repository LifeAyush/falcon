package health

import (
	"net/http"
	"net/url"
	"time"

	"github.com/LifeAyush/falcon/pkg/balancer"
)

// ProbeGET hits path on the server URL and returns whether the backend looks healthy.
func ProbeGET(client *http.Client, base *url.URL, path string) bool {
	if base == nil || path == "" {
		return true
	}
	u := *base
	u.Path = path
	u.RawQuery = ""
	resp, err := client.Get(u.String())
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// StartProbes periodically marks backends dead or alive based on healthPath.
func StartProbes(servers []*balancer.Server, interval, timeout time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			for _, s := range servers {
				path := s.HealthPath
				if path == "" {
					continue
				}
				ok := ProbeGET(client, s.GetURL(), path)
				s.SetDead(!ok)
			}
		}
	}()
}
