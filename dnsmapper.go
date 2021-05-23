package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/devel/dnsmapper/storeapi"
	lru "github.com/hashicorp/golang-lru"
	"github.com/miekg/dns"
)

// Current version
var VERSION = "2.9.3"

var (
	flagdomain     = flag.String("domain", "example.com", "base domain for the dnsmapper")
	flagip         = flag.String("ip", "127.0.0.1", "set the IP address")
	flagdnsport    = flag.Int("dnsport", 53, "Set the DNS port")
	flaghttpport   = flag.Int("httpport", 80, "Set the HTTP port")
	flaghttpsport  = flag.Int("httpsport", 443, "Set the HTTP/TLS port")
	flagtlskeyfile = flag.String("tlskeyfile", "", "Specify path to TLS key (optional)")
	flagtlscrtfile = flag.String("tlscertfile", "", "Specify path to TLS certificate (optional)")

	flagacmedomain = flag.String("acmedomain", "", "Domain to cname _acme-challenge.${domain} to")

	flaglog        = flag.Bool("log", false, "be more verbose")
	flagreporthost = flag.String("reporthost", "", "Hostname for results host")

	flagPrimaryNs = flag.String("ns", "ns.example.com", "nameserver names (comma separated)")
)

type logChannel chan *storeapi.RequestData

var baseLength int
var primaryNsList []string

var cache *lru.Cache

var ch logChannel

func getUUIDFromDomain(name string) string {
	lx := dns.SplitDomainName(name)
	if len(lx) <= baseLength {
		return ""
	}
	ql := lx[0 : len(lx)-baseLength]
	return strings.ToLower(strings.Join(ql, "."))
}

func setup() {
	baseLength = dns.CountLabel(*flagdomain)

	primaryNsList = strings.Split(*flagPrimaryNs, ",")

	log.Println("Listening for requests to", *flagdomain)
}

func init() {
	flag.Parse()
	ch = make(logChannel, posterCount*20)

	os.Setenv("PGSSLMODE", "disable")

	// Setup a cache to support X DNS requests per time-period
	// per server where time-period is how long the client
	// gets to come back with http after doing the DNS request.
	var err error
	cache, err = lru.New(20000)
	if err != nil {
		log.Fatalf("Could not setup lru cache: %s", err)
	}

}

func main() {
	log.Printf("Starting dnsmapper %s\n", VERSION)

	runtime.MemProfileRate = 1

	setup()

	dns.HandleFunc(*flagdomain, setupServerFunc())

	for i := 0; i < posterCount; i++ {
		go reportPoster(ch)
	}

	go httpHandler(*flagip, *flaghttpport, *flaghttpsport)
	go listenAndServeDNS(*flagip, *flagdnsport)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)

	<-terminate
	log.Printf("dnsmapper: signal received, stopping")

	f, err := os.Create("dnsmapper.pprof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.WriteHeapProfile(f)
	err = f.Close()
	if err != nil {
		log.Println("Error closing profile:", err)
	}
	log.Println("... exiting.")

}
