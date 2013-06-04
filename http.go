package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

type ipResponse struct {
	DNS  string
	EDNS string
	HTTP string
}

func uuid() string {
	buf := make([]byte, 16)
	io.ReadFull(rand.Reader, buf)
	return fmt.Sprintf("%x", buf)
}

func jsonData(req *http.Request) (string, error) {
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)

	resp := &ipResponse{HTTP: ip, DNS: ""}

	uuid := getUuidFromDomain(req.Host)
	get := Redis.Get("dns-" + uuid)
	if err := get.Err(); err != nil {
		return "", errors.New("UUID not found")
	}

	resp.DNS = get.Val()

	get = Redis.Get("dnsedns-" + uuid)
	if err := get.Err(); err == nil {
		resp.EDNS = get.Val()
	}

	js, err := json.Marshal(resp)
	if err != nil {
		log.Print("JSON ERROR:", err)
		return "", err
	}
	return string(js), nil
}

func redirectUuid(w http.ResponseWriter, req *http.Request) {
	uuid := uuid()
	host := uuid + "." + *flagdomain
	http.Redirect(w, req, "http://"+host+req.RequestURI, 302)
	return
}

func mainServer(w http.ResponseWriter, req *http.Request) {

	log.Println("HTTP request from", req.RemoteAddr, req.Host)

	uuid := getUuidFromDomain(req.Host)
	if uuid == "www" {
		redirectUuid(w, req)
		return
	}

	if req.URL.Path == "/jsonp" || req.URL.Path == "/json" {

		js, err := jsonData(req)
		if err != nil {
			redirectUuid(w, req)
			return
		}

		if jsonp := req.FormValue("jsonp"); len(jsonp) > 0 {
			io.WriteString(w, jsonp+"("+js+");\n")
			return
		}

		// not jsonp
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
	http.HandleFunc("/", mainServer)

	log.Fatal(http.ListenAndServe(*flagip+":"+*flaghttpport, nil))
}
