package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/devel/dnsmapper/storeapi"
	"github.com/kr/pretty"
	_ "github.com/lib/pq"
	"github.com/oschwald/geoip2-golang"
)

var (
	listen    = flag.String("listen", "", "Listen on this ip:port for the HTTP API")
	geoipPath = flag.String("geoip", "", "Optional directory for geoip database files")
	dbuser    = flag.String("dbuser", "ask", "Postgres user name")
	dbpass    = flag.String("dbpass", "", "Postgres password")
	dbhost    = flag.String("dbhost", "localhost", "Postgres host name")
)

var (
	geodb  *geoip2.Reader
	geoasn *geoip2.Reader
	db     *sql.DB
)

func init() {
	flag.Parse()

	path, _ := os.LookupEnv("GEOIP")

	if len(*geoipPath) > 0 {
		path = *geoipPath
	}

	var err error

	dbName := "GeoIP2-City.mmdb"

	if len(path) > 0 {
		dbName = filepath.Join(path, dbName)
	}

	geodb, err = geoip2.Open(dbName)
	if err != nil {
		log.Printf("Could not open GeoIP database '%s': %s", dbName, err)
		log.Fatal(err)
	}
	log.Printf("Opened '%s' (%s)", dbName, geodb.Metadata().DatabaseType)

	// geoasn, err = geoip.OpenType(geoip.GEOIP_ASNUM_EDITION)
	// if err != nil {
	// 	log.Fatalf("Could not open asn geoip database: %s", err)
	// }
}

func main() {
	startHttp(*listen)
}

func buildMux() *http.ServeMux {

	mux := http.NewServeMux()

	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)

	router, err := rest.MakeRouter(
		rest.Post("/api/v1/store-result", storeHandler),
	)
	if err != nil {
		log.Fatalf("Could not configure router: %s", err)
	}

	api.SetApp(router)

	mux.Handle("/api/v1/", api.MakeHandler())

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok\n"))
		return
	})

	return mux

}

func startHttp(listen string) {
	fmt.Printf("Listening on http://%s\n", listen)
	err := http.ListenAndServe(listen, buildMux())
	fmt.Printf("Could not listen to %s: %s", listen, err)
}

func storeHandler(w rest.ResponseWriter, r *rest.Request) {

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

	data.ClientCC, data.ClientRC, data.ClientASN = ccLookup(net.ParseIP(data.ClientIP))
	data.ServerCC, data.ServerRC, data.ServerASN = ccLookup(net.ParseIP(data.ServerIP))

	if len(data.EdnsNet) > 0 {
		ednsIP, _, _ := net.ParseCIDR(data.EdnsNet)
		data.EdnsCC, data.EdnsRC, data.EdnsASN = ccLookup(net.ParseIP(ednsIP.String()))
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

func ccLookup(ip net.IP) (string, string, int) {
	if ip == nil {
		return "", "", 0
	}

	// If you are using strings that may be invalid, check that ip is not nil
	record, err := geodb.City(ip)

	if err != nil || record.Country.IsoCode == "" {
		if err == nil {
			err = errors.New("not found")
		}
		log.Printf("Could not lookup data for '%s': %s", ip.String(), err)
		return "", "", 0
	}
	fmt.Printf("city name: %v\n", record.City.Names["en"])
	if len(record.Subdivisions) > 0 {
		fmt.Printf("subdivision name: %v\n", record.Subdivisions[0].Names["en"])
	}
	fmt.Printf("country name: %v\n", record.Country.Names["en"])
	fmt.Printf("ISO country code: %v\n", record.Country.IsoCode)
	fmt.Printf("Time zone: %v\n", record.Location.TimeZone)
	fmt.Printf("Coordinates: %v, %v\n", record.Location.Latitude, record.Location.Longitude)

	pretty.Println(record)

	asn := 0

	// asnStr, _ := geoasn.GetName(ip)
	// if len(asnStr) > 0 {
	// 	spaceIdx := strings.Index(asnStr, " ")
	// 	if spaceIdx > 0 {
	// 		asnStr = asnStr[2:spaceIdx]
	// 		asn, _ = strconv.Atoi(asnStr)
	// 	}
	// }

	region := ""
	if len(record.Subdivisions) > 0 {
		region = record.Subdivisions[0].IsoCode
	}

	return record.Country.IsoCode, region, asn
}

func dbConnect() {
	var err error
	db, err = sql.Open("postgres", fmt.Sprintf("user=%s host=%s password=%s sslmode=disable", *dbuser, *dbhost, *dbpass))
	if err != nil {
		log.Fatalf("create db error: %s", err)
	}
	db.SetMaxOpenConns(50)
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

	if err != nil {
		fmt.Printf("DB Error: %s\n", err)
		return err
	}

	updated, _ := rv.RowsAffected()
	fmt.Printf("Updated %d rows\n", updated)

	return err
}
