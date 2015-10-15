package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/devel/dnsmapper/storeapi"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	listen = flag.String("listen", "", "Listen on this ip:port for the HTTP API")
	dbuser = flag.String("dbuser", "ask", "Postgres user name")
	dbpass = flag.String("dbpass", "", "Postgres password")
	dbhost = flag.String("dbhost", "localhost", "Postgres host name")
)

var (
	db        *sqlx.DB
	localNets []*net.IPNet
)

func init() {
	flag.Parse()

	pn := []string{"10.0.0.0/8", "192.168.0.0/16"}
	for _, p := range pn {
		_, ipnet, err := net.ParseCIDR(p)
		if err != nil {
			panic(err)
		}
		localNets = append(localNets, ipnet)
	}
}

func main() {
	startHttp(*listen)
}

func buildMux() *http.ServeMux {

	mux := http.NewServeMux()

	m := martini.Classic()
	m.Use(gzip.All())
	m.Use(render.Renderer())
	m.Get("/api/v1/myip", myIpHandler)

	mux.Handle("/", m)

	return mux

}

func startHttp(listen string) {
	fmt.Printf("Listening on http://%s\n", listen)
	err := http.ListenAndServe(listen, buildMux())
	fmt.Printf("Could not listen to %s: %s", listen, err)
}

func myIpHandler(res http.ResponseWriter, r *http.Request, rndr render.Render) string {

	if db == nil {
		dbConnect()
	}

	// now := time.Now().UTC()

	ipStr := remoteIP(r)
	log.Println("getting where IP is", ipStr)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Printf("Not a valid IP address (X-Forwarded-For) '%s'", ipStr)
		res.WriteHeader(400)
		return ""
	}

	ips := []storeapi.LogData{}
	err := db.Select(&ips, "SELECT * FROM ips where client_ip = $1 order by last_seen desc", ip.String())
	if err != nil {
		log.Fatalf("query err: %s", err)
	}

	res.Header().Set("Cache-Control", "private, must-revalidate, max-age=0")

	rndr.JSON(200, ips)
	return ""

}

func dbConnect() {
	var err error
	db, err = sqlx.Connect("postgres", fmt.Sprintf("user=%s host=%s password=%s sslmode=disable", *dbuser, *dbhost, *dbpass))
	if err != nil {
		log.Fatalf("create db error: %s", err)
	}
	db.SetMaxOpenConns(50)
}

func localNet(ip net.IP) bool {
	for _, n := range localNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if len(xff) > 0 {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			nip := net.ParseIP(ip)
			if nip != nil {
				if localNet(nip) {
					continue
				}
				return nip.String()
			}
		}
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
