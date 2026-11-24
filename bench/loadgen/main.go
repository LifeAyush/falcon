// Command loadgen drives a closed-loop HTTP load test: c workers each send
// requests back to back over keep-alive connections for a fixed duration,
// then it reports throughput and latency percentiles.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type result struct {
	lat  []time.Duration
	errs int
}

func main() {
	url := flag.String("url", "http://127.0.0.1:8100/", "target URL")
	conc := flag.Int("c", 50, "concurrent workers")
	dur := flag.Duration("d", 10*time.Second, "measured duration")
	warm := flag.Duration("warmup", 2*time.Second, "warmup before measuring")
	label := flag.String("label", "", "row label for markdown output")
	flag.Parse()

	tr := &http.Transport{
		MaxIdleConns:        *conc * 2,
		MaxIdleConnsPerHost: *conc * 2,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	run := func(d time.Duration, record bool) []result {
		res := make([]result, *conc)
		deadline := time.Now().Add(d)
		var wg sync.WaitGroup
		for i := 0; i < *conc; i++ {
			wg.Add(1)
			go func(r *result) {
				defer wg.Done()
				for time.Now().Before(deadline) {
					start := time.Now()
					resp, err := client.Get(*url)
					if err == nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
						if resp.StatusCode != http.StatusOK {
							err = fmt.Errorf("status %d", resp.StatusCode)
						}
					}
					if !record {
						continue
					}
					if err != nil {
						r.errs++
						continue
					}
					r.lat = append(r.lat, time.Since(start))
				}
			}(&res[i])
		}
		wg.Wait()
		return res
	}

	run(*warm, false)
	res := run(*dur, true)

	var all []time.Duration
	errs := 0
	for _, r := range res {
		all = append(all, r.lat...)
		errs += r.errs
	}
	if len(all) == 0 {
		fmt.Fprintln(os.Stderr, "no successful requests")
		os.Exit(1)
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	pct := func(p float64) time.Duration { return all[int(float64(len(all)-1)*p)] }
	rps := float64(len(all)) / dur.Seconds()
	ms := func(d time.Duration) string { return fmt.Sprintf("%.2f", float64(d.Microseconds())/1000) }

	// | label | conc | rps | p50 | p90 | p99 | max | errors |
	fmt.Printf("| %s | %d | %.0f | %s | %s | %s | %s | %d |\n",
		*label, *conc, rps, ms(pct(0.50)), ms(pct(0.90)), ms(pct(0.99)), ms(all[len(all)-1]), errs)
}
