package main

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"github.com/devel/dnsmapper/storeapi"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type ipResponse struct {
	DNS  string
	EDNS string
	HTTP string
}

var uuidCh chan string

func uuidFactory() {
	uuidCh = make(chan string, 100)

	enc := base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")

	length := 20

	buf := make([]byte, length)
	uuid := make([]byte, enc.EncodedLen(length))

	for {
		rand.Read(buf)
		enc.Encode(uuid, buf)
		uuidCh <- string(uuid)
	}
}

func uuid() string {
	return <-uuidCh
}

func jsonData(req *http.Request) (string, error) {
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)

	resp := &ipResponse{HTTP: ip, DNS: ""}

	uuid := getUuidFromDomain(req.Host)
	get := Redis.Get("dns-" + uuid)
	if err := get.Err(); err != nil {
		return "", errors.New("UUID not found")
	}

	v := strings.Split(get.Val(), " ")

	resp.DNS = v[0]
	if len(v) > 1 {
		resp.EDNS = v[1]
	}

	js, err := json.Marshal(resp)
	if err != nil {
		log.Print("JSON ERROR:", err)
		return "", err
	}

	data := storeapi.RequestData{
		TestIP:   *flagip,
		ServerIP: resp.DNS,
		ClientIP: resp.HTTP,
		EnumNet:  resp.EDNS,
	}
	select {
	case ch <- &data:
	default:
		log.Println("dropped log data, queue full")
	}

	return string(js), nil
}

func redirectUuid(w http.ResponseWriter, req *http.Request) {
	uuid := uuid()
	host := uuid + "." + *flagdomain

	proto := "http"

	if req.TLS != nil {
		proto = "https"
	}

	http.Redirect(w, req, proto+"://"+host+req.RequestURI, 302)
	return
}

func mainServer(w http.ResponseWriter, req *http.Request) {

	log.Println("HTTP request from", req.RemoteAddr, req.Host)

	if req.URL.Path == "/jsonp" || req.URL.Path == "/json" || req.URL.Path == "/none" {

		w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")

		uuid := getUuidFromDomain(req.Host)
		if uuid == "www" {
			redirectUuid(w, req)
			return
		}

		js, err := jsonData(req)
		if err != nil {
			redirectUuid(w, req)
			return
		}

		if req.URL.Path == "/none" {
			w.WriteHeader(204)
			return
		}

		if jsonp := req.FormValue("jsonp"); len(jsonp) > 0 {
			w.Header().Set("Content-Type", "text/javascript")
			io.WriteString(w, jsonp+"("+js+");\n")
			return
		}

		// not jsonp
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, js+"\n")
		return

	}

	if req.URL.Path == "/version" {
		io.WriteString(w, `<html><head><title>DNS Mapper `+
			VERSION+`</title><body>`+
			`Hello`+
			`</body></html>`)
		return
	}

	http.NotFound(w, req)
	return
}

func httpHandler() {

	go uuidFactory()

	http.HandleFunc("/", mainServer)

	if len(*flagtlskeyfile) > 0 {

		log.Printf("Starting TLS with key='%s' and cert='%s'",
			*flagtlskeyfile,
			*flagtlscrtfile,
		)

		go func() {
			tlslisten := *flagip + ":" + *flaghttpsport
			log.Println("Going to listen for TLS requests on port", tlslisten)
			log.Fatal(http.ListenAndServeTLS(
				tlslisten,
				*flagtlscrtfile,
				*flagtlskeyfile,
				nil,
			))
		}()
	}

	listen := *flagip + ":" + *flaghttpport
	log.Println("HTTP listen on", listen)
	log.Fatal(http.ListenAndServe(listen, nil))

}
