package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/miekg/dns"
	librato "github.com/rcrowley/go-librato"
)

var (
	m            librato.Metrics
	pollInterval = 60
	dialTimeout  = 5
	debug        = false
)

type result struct {
	ip  string
	nss []string
	ok  bool
	err error
}

type ipSet struct {
	m map[string]map[string]bool
}

func newIPSet() ipSet {
	return ipSet{
		m: make(map[string]map[string]bool),
	}
}

func (ips *ipSet) addIP(ip string, ns string) {
	if ips.m[ip] == nil {
		ips.m[ip] = make(map[string]bool)
	}
	ips.m[ip][ns] = true
}

func (ips *ipSet) getIPToNSMap() map[string][]string {
	out := make(map[string][]string)
	for ip, nsm := range ips.m {
		for ns, _ := range nsm {
			out[ip] = append(out[ip], ns)
		}
	}
	return out
}

func (ips *ipSet) getIPs() []string {
	var out []string
	for ip, _ := range ips.m {
		out = append(out, ip)
	}
	return out
}

func resolveHostname(hostname string) (ipSet, error) {
	ips := newIPSet()

	c := dns.Client{}

	m := dns.Msg{}
	m.SetQuestion(hostname+".", dns.TypeNS)
	r, _, err := c.Exchange(&m, "8.8.8.8:53")
	if err != nil {
		return ips, err
	}

	for _, ans := range r.Answer {
		nsRecord := ans.(*dns.NS)
		server := nsRecord.Ns

		m2 := dns.Msg{}
		m2.SetQuestion(hostname+".", dns.TypeA)
		r2, _, err := c.Exchange(&m2, server+":53")
		if err != nil {
			return ips, err
		}

		for _, ans := range r2.Answer {
			aRecord := ans.(*dns.A)
			ips.addIP(aRecord.A.String(), server)
		}
	}

	return ips, nil
}

func checkHostHealth(hostname string) ([]result, error) {
	ips, err := resolveHostname(hostname)
	if err != nil {
		return []result{}, err
	}

	out := make(chan result)

	for ip, nss := range ips.getIPToNSMap() {
		go func(ip string, nss []string) {
			conn, err := net.DialTimeout(
				"tcp",
				ip+":443",
				time.Duration(dialTimeout)*time.Second,
			)
			if conn != nil {
				conn.Close()
			}
			out <- result{
				ip:  ip,
				nss: nss,
				ok:  err == nil,
				err: err,
			}
		}(ip, nss)
	}

	var res []result
	for _ = range ips.getIPs() {
		res = append(res, <-out)
	}

	return res, nil
}

func metricsify(s string) string {
	return strings.Replace(s, ".", "_", -1)
}

func processError(err error) {
	log.Printf("error: %v", err)
	raven.CaptureErrorAndWait(err, nil)
}

func runMonitor(hostname string) {
	for {
		log.Print("polling " + hostname)

		res, err := checkHostHealth(hostname)

		if err != nil {
			processError(err)
			continue
		}

		numBorked := 0
		for _, r := range res {
			if debug {
				log.Printf("ok=%v err=%v ip=%v nss=%v", r.ok, r.err, r.ip, r.nss)
			}

			if !r.ok {
				numBorked++
				log.Printf("borked ip %v with error %v and nss %v", r.ip, r.err, r.nss)
				processError(r.err)
			}
		}

		if m != nil {
			g := m.GetGauge("travis.lb-monitor." + metricsify(hostname))
			g <- int64(numBorked)
		}

		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
}

func main() {
	if os.Getenv("HOSTNAMES") == "" {
		log.Fatal("please provide the HOSTNAMES env variable")
	}

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

	if os.Getenv("DIAL_TIMEOUT") != "" {
		dialTimeout, err = strconv.Atoi(os.Getenv("DIAL_TIMEOUT"))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("running with DIAL_TIMEOUT of %v", dialTimeout)
	} else {
		log.Printf("defaulting DIAL_TIMEOUT to %v", dialTimeout)
	}

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

	if os.Getenv("SENRTY_DSN") != "" {
		err := raven.SetDSN(os.Getenv("SENRTY_DSN"))
		if err != nil {
			log.Fatal(err)
		}

		// TODO: raven.SetRelease(VersionString)
		if os.Getenv("SENRTY_ENVIRONMENT") != "" {
			raven.SetEnvironment(os.Getenv("SENRTY_ENVIRONMENT"))
		}
	}

	hostnames := strings.Split(os.Getenv("HOSTNAMES"), ",")
	debug = os.Getenv("DEBUG") == "true"

	for _, hostname := range hostnames {
		go func(hostname string) {
			raven.CapturePanicAndWait(func() {
				runMonitor(hostname)
			}, nil)
		}(hostname)
	}

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}
