package main

import (
	"flag"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/devel/dnsmapper/storeapi"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net"
	"net/http"
)

var (
	listen = flag.String("listen", "", "Listen on this ip:port for the HTTP API")
	dbuser = flag.String("dbuser", "ask", "Postgres user name")
	dbpass = flag.String("dbpass", "", "Postgres password")
	dbhost = flag.String("dbhost", "localhost", "Postgres host name")
)

var (
	db *sqlx.DB
)

func init() {
	flag.Parse()
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

	ipStr := r.Header.Get("X-Forwarded-For")
	if len(ipStr) == 0 {
		log.Println("Didn't get X-Forwarded-for!")
		ipStr = r.RemoteAddr
	}

	log.Println("getting where IP is", ipStr)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		res.WriteHeader(400)
		return ""
	}

	ips := []storeapi.LogData{}
	err := db.Selectv(&ips, "SELECT * FROM ips where client_ip = $1 order by last_seen desc", ip.String())
	if err != nil {
		log.Fatalf("query err: %s", err)
	}

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
