package storeapi

import (
	"encoding/json"
	"time"
)

type RequestData struct {
	ClientIP string
	ServerIP string
	EdnsNet  string
	TestIP   string
}

type LogData struct {
	ClientIP  string     `db:"client_ip"`
	ServerIP  string     `db:"server_ip"`
	EdnsNet   string     `db:"edns_net"`
	ClientCC  string     `db:"client_cc"`
	ClientRC  string     `db:"client_rc"`
	ServerCC  string     `db:"server_cc"`
	ServerRC  string     `db:"server_rc"`
	EdnsCC    string     `db:"edns_cc"`
	EdnsRC    string     `db:"edns_rc"`
	ClientASN uint       `db:"client_asn"`
	ServerASN uint       `db:"server_asn"`
	EdnsASN   uint       `db:"edns_asn"`
	HasEdns   bool       `db:"has_edns"`
	TestIP    string     `db:"test_ip" json:"-"`
	FirstSeen *time.Time `db:"first_seen" json:"-"`
	LastSeen  *time.Time `db:"last_seen" json:"-"`
}

func (data *RequestData) JSON() ([]byte, error) {
	return json.Marshal(data)
}
