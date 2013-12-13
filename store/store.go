package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/abh/geoip"
	"github.com/ant0ine/go-json-rest"
	"github.com/devel/dnsmapper/storeapi"
	_ "github.com/lib/pq"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	listen    = flag.String("listen", "", "Listen on this ip:port for the HTTP API")
	geoipPath = flag.String("geoip", "", "Optional directory for geoip database files")
	dbuser    = flag.String("dbuser", "ask", "Postgres user name")
	dbhost    = flag.String("dbhost", "localhost", "Postgres host name")
)

var (
	geodb  *geoip.GeoIP
	geoasn *geoip.GeoIP
	db     *sql.DB
)

func init() {
	flag.Parse()

	if len(*geoipPath) > 0 {
		geoip.SetCustomDirectory(*geoipPath)
	}

	var err error
	geodb, err = geoip.OpenType(geoip.GEOIP_CITY_EDITION_REV1)
	if err != nil {
		log.Fatalf("Could not open city geoip database: %s", err)
	}

	geoasn, err = geoip.OpenType(geoip.GEOIP_ASNUM_EDITION)
	if err != nil {
		log.Fatalf("Could not open asn geoip database: %s", err)
	}
}

func main() {
	startHttp(*listen)
}

func buildMux() *http.ServeMux {

	mux := http.NewServeMux()

	restHandler := rest.ResourceHandler{}
	restHandler.EnableGzip = true
	restHandler.EnableLogAsJson = true
	restHandler.EnableResponseStackTrace = true
	//restHandler.EnableStatusService = true

	restHandler.SetRoutes(
		rest.Route{"POST", "/api/v1/store-result", storeHandler},
	)

	mux.Handle("/api/v1/", &restHandler)

	return mux

}

func startHttp(listen string) {
	fmt.Printf("Listening on http://%s\n", listen)
	err := http.ListenAndServe(listen, buildMux())
	fmt.Printf("Could not listen to %s: %s", listen, err)
}

func storeHandler(w *rest.ResponseWriter, r *rest.Request) {

	// err := r.ParseForm()
	// if err != nil {
	// log.Println("parse error", err)
	// }

	now := time.Now().UTC()

	reqData := &storeapi.RequestData{}
	err := r.DecodeJsonPayload(&reqData)
	if err != nil {
		rest.Error(w, "Could not decode JSON: "+err.Error(), 400)
		return
	}

	data := &storeapi.LogData{
		TestIP:   reqData.TestIP,
		ClientIP: reqData.ClientIP,
		ServerIP: reqData.ServerIP,
		EdnsNet:  reqData.EdnsNet,
		LastSeen: &now,
	}

	data.ClientCC, data.ClientRC, data.ClientASN = ccLookup(data.ClientIP)
	data.ServerCC, data.ServerRC, data.ServerASN = ccLookup(data.ServerIP)

	if len(data.EdnsNet) > 0 {
		ednsIP, _, _ := net.ParseCIDR(data.EdnsNet)
		data.EdnsCC, data.EdnsRC, data.EdnsASN = ccLookup(ednsIP.String())
		data.HasEdns = true
	} else {
		data.EdnsNet = data.ServerIP
		data.EdnsCC, data.EdnsRC, data.EdnsASN = data.ServerCC, data.ServerRC, data.ServerASN
		data.HasEdns = false
	}

	w.WriteHeader(204)

	// w.WriteJson(data)

	dbStore(data)

}

func ccLookup(ip string) (string, string, int) {
	if len(ip) == 0 {
		return "", "", 0
	}

	r := geodb.GetRecord(ip)
	if r == nil {
		return "", "", 0
	}

	var asn int

	asnStr, _ := geoasn.GetName(ip)
	if len(asnStr) > 0 {
		spaceIdx := strings.Index(asnStr, " ")
		if spaceIdx > 0 {
			asnStr = asnStr[2:spaceIdx]
			asn, _ = strconv.Atoi(asnStr)
		}
	}

	return r.CountryCode, r.Region, asn
}

func dbConnect() {
	var err error
	db, err = sql.Open("postgres", fmt.Sprintf("user=%s host=%s sslmode=disable", *dbuser, *dbhost))
	if err != nil {
		log.Fatalf("create db error: %s", err)
	}
}

func dbStore(data *storeapi.LogData) error {

	if db == nil {
		dbConnect()
	}

	fmt.Printf("dbStore: %#v\n", data)

	rv, err := db.Exec(`
	WITH upsert_data AS (
	    SELECT
    	$1::inet AS client_ip,
    	$2::inet AS server_ip,
    	$3::cidr AS edns_net,

    	$4::char(2) AS client_cc,
    	$5::char(2) AS client_rc,
    	$6::int AS client_asn,

    	$7::char(2) AS server_cc,
    	$8::char(2) AS server_rc,
    	$9::int AS     server_asn,

    	$10::char(2) AS edns_cc,
    	$11::char(2) AS edns_rc,
    	$12::int AS     edns_asn,

    	$13::inet AS test_ip,

    	$14::boolean AS has_edns,
    	$15::timestamp AS last_seen
	),
	update_ips AS (
    	UPDATE ips
    	SET
    		client_cc = ud.client_cc,
    		client_rc = ud.client_rc,
    		client_asn = ud.client_asn,

    		server_cc = ud.server_cc,
    		server_rc = ud.server_rc,
    		server_asn = ud.server_asn,

    		edns_cc = ud.edns_cc,
    		edns_rc = ud.edns_rc,
    		edns_asn = ud.edns_asn,

    		test_ip = ud.test_ip,

    		has_edns = ud.has_edns,
    		last_seen = ud.last_seen

    	FROM upsert_data ud
    	WHERE
    		ips.client_ip = ud.client_ip AND
    		ips.server_ip = ud.server_ip
	    RETURNING ips.*
	)
    INSERT INTO
        ips
		(client_ip, server_ip, edns_net,
		 client_cc, client_rc, client_asn,
		 server_cc, server_rc, server_asn,
		 edns_cc, edns_rc, edns_asn,
		 test_ip, has_edns,
		 first_seen, last_seen
		)
		SELECT
			client_ip, server_ip, edns_net,
			client_cc, client_rc, client_asn,
			server_cc, server_rc, server_asn,
			edns_cc, edns_rc, edns_asn,
			test_ip, has_edns,
			last_seen, last_seen
			FROM upsert_data
			WHERE NOT EXISTS (
				SELECT 1 FROM update_ips up
				WHERE (up.client_ip = upsert_data.client_ip)
			)
	`,
		data.ClientIP, data.ServerIP, data.EdnsNet,
		data.ClientCC, data.ClientRC, data.ClientASN,
		data.ServerCC, data.ServerRC, data.ServerASN,
		data.EdnsCC, data.EdnsRC, data.EdnsASN,
		data.TestIP, data.HasEdns, data.LastSeen,
	)

	fmt.Printf("DB RESULT: %#v: %s\n", rv, err)

	return err
}

func queryExp() {

	serverIp := "8.8.8.8"
	rows, err := db.Query("SELECT client_ip from ips where server_ip = $1", serverIp)

	if err != nil {
		log.Fatalf("query err: %s", err)
	}

	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)
		if err != nil {
			log.Fatalf("Scan error: %s", err)
		}

		log.Println("got ip", ip)

	}
	err = rows.Err() // get any error encountered during iteration
	if err != nil {
		log.Fatalf("Scan error: %s", err)
	}
}
