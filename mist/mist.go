package main

//go:generate esc -o static.go -ignore .DS_Store public

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/ant0ine/go-json-rest/rest"

	"github.com/devel/dnsmapper/storeapi"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	listen = flag.String("listen", "", "Listen on this ip:port for the HTTP API")
	dbuser = flag.String("dbuser", "ask", "Postgres user name")
	dbpass = flag.String("dbpass", "", "Postgres password")
	dbhost = flag.String("dbhost", "localhost", "Postgres host name")
	devel  = flag.Bool("devel", false, "development mode")
	// todo: make schema configurable
)

var (
	db        *sqlx.DB
	localNets []*net.IPNet
)

func init() {
	flag.Parse()

	os.Setenv("PGSSLMODE", "disable")

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
	startHTTP(*listen)
}

func buildMux() *http.ServeMux {

	mux := http.NewServeMux()

	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)

	router, err := rest.MakeRouter(
		rest.Get("/myip", myIPHandler),
	)
	if err != nil {
		log.Fatalf("Could not configure router: %s", err)
	}

	api.SetApp(router)

	mux.Handle("/", http.FileServer(Dir(*devel, "/public")))

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api.MakeHandler()))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok\n"))
		return
	})

	return mux
}

func startHTTP(listen string) {
	fmt.Printf("Listening on http://%s\n", listen)
	err := http.ListenAndServe(listen, buildMux())
	fmt.Printf("Could not listen to %s: %s", listen, err)
}

func myIPHandler(w rest.ResponseWriter, r *rest.Request) {

	if db == nil {
		err := dbConnect()
		if err != nil {
			http.Error(w.(http.ResponseWriter), "db connect error", 500)
		}
	}

	// now := time.Now().UTC()

	ipStr := remoteIP(r.Header)
	if len(ipStr) == 0 {
		ipStr, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	log.Println("getting where IP is", ipStr)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Printf("Not a valid IP address (X-Forwarded-For) '%s'", ipStr)
		http.Error(w.(http.ResponseWriter), "Invalid IP", 400)
		return
	}

	ips := []storeapi.LogData{}
	err := db.Select(&ips, "SELECT * FROM ips where client_ip = $1 order by last_seen desc", ip.String())
	if err != nil {
		log.Printf("query err: %s", err)
		http.Error(w.(http.ResponseWriter), "db error", 500)
	}

	w.Header().Set("Cache-Control", "private, must-revalidate, max-age=0")

	w.WriteJson(ips)

	return
}

func dbConnect() error {
	var err error
	db, err = sqlx.Connect("postgres", fmt.Sprintf("user=%s host=%s password=%s sslmode=disable", *dbuser, *dbhost, *dbpass))
	if err != nil {
		return err
	}
	db.Exec("SET search_path TO dnsmapper,public")
	db.SetMaxOpenConns(50)
	return db.Ping()
}

func localNet(ip net.IP) bool {
	for _, n := range localNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteIP(h http.Header) string {
	xff := h.Get("X-Forwarded-For")
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

	return ""
}
