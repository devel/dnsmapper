package main

import (
	"flag"
	"github.com/abh/dns"
	"github.com/devel/dnsmapper/storeapi"
	"log"
	"os"
	"os/signal"
	"strings"
)

var VERSION = "2.2.1"

var (
	flagdomain     = flag.String("domain", "example.com", "base domain for the dnsmapper")
	flagip         = flag.String("ip", "127.0.0.1", "set the IP address")
	flagdnsport    = flag.String("dnsport", "53", "Set the DNS port")
	flaghttpport   = flag.String("httpport", "80", "Set the HTTP port")
	flaghttpsport  = flag.String("httpsport", "443", "Set the HTTP/TLS port")
	flagtlskeyfile = flag.String("tlskeyfile", "", "Specify path to TLS key (optional)")
	flagtlscrtfile = flag.String("tlscertfile", "", "Specify path to TLS certificate (optional)")

	flaglog        = flag.Bool("log", false, "be more verbose")
	flagreporthost = flag.String("reporthost", "", "Hostname for results host")

	flagPrimaryNs = flag.String("ns", "ns.example.com", "nameserver names (comma separated)")
)

type logChannel chan *storeapi.RequestData

var baseLength int
var primaryNsList []string

var ch logChannel

func getUuidFromDomain(name string) string {
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
}

func main() {
	log.Printf("Starting dnsmapper %s\n", VERSION)

	setup()

	dns.HandleFunc(*flagdomain, setupServerFunc())

	redisConnect()

	for i := 0; i < posterCount; i++ {
		go reportPoster(ch)
	}

	go httpHandler()
	go listenAndServeDNS(*flagip + ":" + *flagdnsport)

	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)

	<-terminate
	log.Printf("dnsmapper: signal received, stopping")

}
