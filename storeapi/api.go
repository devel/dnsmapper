package storeapi

import (
	"encoding/json"
	"time"
)

type RequestData struct {
	ClientIP string
	ServerIP string
	EnumNet  string
	TestIP   string
}

type LogData struct {
	ClientIP  string
	ServerIP  string
	EnumNet   string
	ClientCC  string
	ClientRC  string
	ServerCC  string
	ServerRC  string
	EnumCC    string
	EnumRC    string
	ClientASN int
	ServerASN int
	EnumASN   int
	HasEnum   bool
	TestIP    string
	FirstSeen *time.Time
	LastSeen  *time.Time
}

func (data *RequestData) JSON() ([]byte, error) {
	return json.Marshal(data)
}
