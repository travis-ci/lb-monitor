package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	librato "github.com/rcrowley/go-librato"
)

type result struct {
	ip  string
	ok  bool
	err error
}

func runMonitor(hostname string, dialTimeout int) ([]result, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return []result{}, err
	}

	out := make(chan result)

	for _, addr := range addrs {
		go func(addr string) {
			_, err := net.DialTimeout(
				"tcp",
				addr+":443",
				time.Duration(dialTimeout)*time.Second,
			)
			out <- result{
				ip:  addr,
				ok:  err == nil,
				err: err,
			}
		}(addr)
	}

	var res []result
	for _ = range addrs {
		res = append(res, <-out)
	}

	return res, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func main() {
	if os.Getenv("HOSTNAME") == "" {
		log.Fatal("please provide the HOSTNAME env variable")
	}

	pollInterval := 60
	var err error
	if os.Getenv("POLL_INTERVAL") != "" {
		pollInterval, err = strconv.Atoi(os.Getenv("POLL_INTERVAL"))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("running with POLL_INTERVAL of %v", pollInterval)
	} else {
		log.Printf("defaulting POLL_INTERVAL to %v", pollInterval)
	}

	dialTimeout := 5
	if os.Getenv("DIAL_TIMEOUT") != "" {
		dialTimeout, err = strconv.Atoi(os.Getenv("DIAL_TIMEOUT"))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("running with DIAL_TIMEOUT of %v", pollInterval)
	} else {
		log.Printf("defaulting DIAL_TIMEOUT to %v", dialTimeout)
	}

	var m librato.Metrics
	if os.Getenv("LIBRATO_USER") != "" && os.Getenv("LIBRATO_TOKEN") != "" {
		source := os.Getenv("LIBRATO_SOURCE")
		if source == "" {
			source = os.Getenv("DYNO")
		}
		if source == "" {
			source, err = os.Hostname()
			if err != nil {
				log.Fatal(err)
			}
		}

		m = librato.NewSimpleMetrics(
			os.Getenv("LIBRATO_USER"),
			os.Getenv("LIBRATO_TOKEN"),
			source,
		)
		defer m.Wait()
		defer m.Close()
	} else {
		log.Print("no librato config provided, to enable librato, please provide LIBRATO_USER and LIBRATO_TOKEN")
	}

	hostname := os.Getenv("HOSTNAME")
	upstreamHostname := os.Getenv("UPSTREAM_HOSTNAME")

	for {
		log.Print("polling " + hostname)

		res, err := runMonitor(hostname, dialTimeout)
		if err != nil {
			log.Print(err)
		}

		var upstreamAddrs []string
		if upstreamHostname != "" {
			upstreamAddrs, err = net.LookupHost(upstreamHostname)
			if err != nil {
				log.Print(err)
			}
		}

		numBorked := 0
		for _, r := range res {
			if !r.ok {
				containedUpstream := ""
				if upstreamHostname != "" {
					if contains(upstreamAddrs, r.ip) {
						containedUpstream = " (contained upstream)"
					} else {
						containedUpstream = " (not contained upstream)"
					}
				}

				numBorked++
				log.Printf("borked ip %v with error %v %v", r.ip, r.err, containedUpstream)
			}
		}

		if m != nil {
			g := m.GetGauge("travis.lb-monitor." + strings.Replace(hostname, ".", "_", -1) + ".borked_ips")
			g <- int64(numBorked)
		}

		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
}
