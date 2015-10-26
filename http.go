package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base32"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/devel/dnsmapper/storeapi"
	"github.com/gorilla/handlers"
)

type ipResponse struct {
	DNS  string
	EDNS string
	HTTP string
}

var (
	uuidCh    chan string
	localNets []*net.IPNet
)

func init() {
	go uuidFactory()

	pn := []string{"10.0.0.0/8", "192.168.0.0/16"}
	for _, p := range pn {
		_, ipnet, err := net.ParseCIDR(p)
		if err != nil {
			panic(err)
		}
		localNets = append(localNets, ipnet)
	}
}

func uuidFactory() {
	uuidCh = make(chan string, 10)

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

	ip := remoteIP(req.Header)
	if len(ip) == 0 {
		ip, _, _ = net.SplitHostPort(req.RemoteAddr)
	}

	resp := &ipResponse{HTTP: ip, DNS: ""}

	uuid := getUUIDFromDomain(req.Host)

	dns, edns, ok := getCache(uuid)

	if !ok {
		return "", errors.New("UUID not found")
	}

	resp.DNS = dns
	resp.EDNS = edns

	js, err := json.Marshal(resp)
	if err != nil {
		log.Print("JSON ERROR:", err)
		return "", err
	}

	data := storeapi.RequestData{
		TestIP:   *flagip,
		ServerIP: resp.DNS,
		ClientIP: resp.HTTP,
		EdnsNet:  resp.EDNS,
	}
	select {
	case ch <- &data:
	default:
		log.Println("dropped log data, queue full")
	}

	return string(js), nil
}

func redirectUUID(w http.ResponseWriter, req *http.Request) {
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

	if req.URL.Path == "/jsonp" || req.URL.Path == "/json" || req.URL.Path == "/none" {

		w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")

		uuid := getUUIDFromDomain(req.Host)
		if uuid == "www" {
			redirectUUID(w, req)
			return
		}

		js, err := jsonData(req)
		if err != nil {
			redirectUUID(w, req)
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

	if req.URL.Path == "/mapper.js" {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.WriteHeader(200)
		io.WriteString(w, `
(function(global){"use strict";var id=function(){var chars="0123456789abcdefghijklmnopqrstuvxyz".split("");
var uuid=[],rnd=Math.random,r;for(var i=0;i<17;i++){if(!uuid[i]){r=0|rnd()*16;uuid[i]=chars[i==19?r&3|8:r&15]}}
return uuid.join("")};
setTimeout(function(){(new Image).src="http://"+id()+".`+
			*flagdomain+
			`/none"},3200)})(this);
`)
		return
	}

	if req.URL.Path == "/" {
		w.Header().Set("Cache-Control", "public, max-age=900")
		w.WriteHeader(200)
		io.WriteString(w, HOMEPAGE)
		return
	}

	if req.URL.Path == "/robots.txt" {
		w.Header().Set("Cache-Control", "public, max-age=604800")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		io.WriteString(w, "# Hi Robot!\n")
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

	if len(*flagtlskeyfile) > 0 {

		log.Printf("Starting TLS with key='%s' and cert='%s'",
			*flagtlskeyfile,
			*flagtlscrtfile,
		)

		go func() {

			tlsconfig := &tls.Config{
				ClientSessionCache: tls.NewLRUClientSessionCache(100),
				MinVersion:         tls.VersionTLS10,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
					tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA},
			}

			tlslisten := *flagip + ":" + *flaghttpsport
			srv := &http.Server{
				Addr:         tlslisten,
				WriteTimeout: 5 * time.Second,
				ReadTimeout:  10 * time.Second,
				TLSConfig:    tlsconfig,
			}
			log.Println("Going to listen for TLS requests on port", tlslisten)
			log.Fatal(srv.ListenAndServeTLS(
				*flagtlscrtfile,
				*flagtlskeyfile,
			))

		}()
	}

	listen := *flagip + ":" + *flaghttpport
	srv := &http.Server{
		Handler:      handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux),
		Addr:         listen,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	log.Println("HTTP listen on", listen)
	log.Fatal(srv.ListenAndServe())

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
