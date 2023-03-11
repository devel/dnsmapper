package main

import (
	"crypto/rand"
	"crypto/tls"
	_ "embed"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
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

//go:embed index.html
var HOMEPAGE string

func init() {
	go uuidFactory()

	pn := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	}
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

func remoteIP(xff string) string {
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

func (resp *ipResponse) JSON() (string, error) {
	js, err := json.Marshal(resp)
	if err != nil {
		log.Print("JSON ERROR:", err)
		return "", err
	}
	return string(js), err
}

func responseData(req *http.Request) (*ipResponse, error) {

	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	nip := net.ParseIP(ip)

	if xff := req.Header.Get("X-Forwarded-For"); len(xff) > 0 && localNet(nip) {
		ip = remoteIP(xff)
	}

	resp := &ipResponse{HTTP: ip, DNS: ""}

	uuid := getUUIDFromDomain(req.Host)

	dns, edns, ok := getCache(uuid)

	if !ok {
		return nil, errors.New("UUID not found")
	}

	resp.DNS = dns
	resp.EDNS = edns

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

	return resp, nil
}

func redirectUUID(w http.ResponseWriter, req *http.Request) {
	uuid := uuid()
	host := uuid + "." + *flagdomain

	proto := "http"

	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		proto = "https"
	}

	http.Redirect(w, req, proto+"://"+host+req.RequestURI, http.StatusFound)
}

var apiPaths = map[string]interface{}{
	"/jsonp":    nil,
	"/json":     nil,
	"/ip":       nil,
	"/none":     nil,
	"/gone":     nil,
	"/notfound": nil,
}

func mainServer(w http.ResponseWriter, req *http.Request) {

	if _, ok := apiPaths[req.URL.Path]; ok {

		w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		uuid := getUUIDFromDomain(req.Host)
		if uuid == "www" {
			redirectUUID(w, req)
			return
		}

		resp, err := responseData(req)
		if err != nil {
			log.Printf("redirecting to new uuid, err: %s", err)
			redirectUUID(w, req)
			return
		}

		switch req.URL.Path {
		case "/none":
			w.WriteHeader(204)
			return
		case "/gone":
			w.WriteHeader(410)
			return
		case "/notfound":
			w.WriteHeader(404)
			return
		case "/ip":
			w.WriteHeader(200)
			w.Write([]byte(resp.HTTP))
			return
		}

		// json request
		js, err := resp.JSON()
		if err != nil {
			w.WriteHeader(500)
			log.Printf("could not convert response %+v to json: %s", resp, err)
			return
		}

		jsonp := req.FormValue("jsonp")
		if len(jsonp) == 0 {
			jsonp = req.FormValue("callback")
		}

		if len(jsonp) > 0 {
			w.Header().Set("Content-Type", "text/javascript")
			io.WriteString(w, jsonp+"("+js+");\n")
			return
		}

		// not jsonp
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, js+"\n")
		return

	}

	mapperScript := `
	 (function(global){"use strict";var id=function(){var chars="0123456789abcdefghijklmnopqrstuvxyz".split("");
	 var uuid=[],rnd=Math.random,r;for(var i=0;i<17;i++){if(!uuid[i]){r=0|rnd()*16;uuid[i]=chars[i==19?r&3|8:r&15]}}
	 return uuid.join("")};
	 setTimeout(function(){(new Image).src=location.protocol+"//"+id()+".` +
		*flagdomain +
		`/none"},3200)})(this);
	 `

	if req.URL.Path == "/mapper.js" {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.WriteHeader(200)
		io.WriteString(w, mapperScript)
		return
	}

	if req.URL.Path == "/mapper-v6compat.js" {
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.WriteHeader(200)
		io.WriteString(w, mapperScript)
		io.WriteString(w, `v6 = { "version": "2", test: function(){} };`)
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
}

func httpListen(h http.Handler, ip string, port int, tlsconfig *tls.Config) error {

	listen := fmt.Sprintf("%s:%d", ip, port)
	srv := &http.Server{
		Handler:      h,
		Addr:         listen,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  10 * time.Second,
		TLSConfig:    tlsconfig,
	}
	if tlsconfig != nil {
		log.Printf("HTTPS listen on %s", listen)
		return srv.ListenAndServeTLS(
			*flagtlscrtfile,
			*flagtlskeyfile,
		)
	}

	log.Printf("HTTP  listen on %s", listen)
	return srv.ListenAndServe()
}

func httpHandler(listenIP string, listenHTTPPort, listenHTTPSPort int) {

	http.HandleFunc("/", mainServer)

	h := handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux)

	if len(*flagtlskeyfile) > 0 {

		log.Printf("Starting TLS with key='%s' and cert='%s'",
			*flagtlskeyfile,
			*flagtlscrtfile,
		)

		tlsconfig := &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
			MinVersion:         tls.VersionTLS10,
		}

		IPs := []string{listenIP}

		// we have some sort of proxy, so listen on localhost
		if listenHTTPSPort != 443 {
			if listenIP != "127.0.0.1" {
				IPs = append(IPs, "127.0.0.1")
			}
		}

		for _, ip := range IPs {
			listenIP := ip
			log.Printf("listenIP TLS: %q", listenIP)
			go func() {
				err := httpListen(h, listenIP, listenHTTPSPort, tlsconfig)
				if err != nil {
					log.Fatalf("https error %s:%d: %s", listenIP, listenHTTPSPort, err)
				}
			}()
		}

	}

	IPs := []string{listenIP}
	if listenHTTPSPort != 80 {
		if listenIP != "127.0.0.1" {
			IPs = append(IPs, "127.0.0.1")
		}
	}

	for _, ip := range IPs {
		listenIP := ip
		go func() {
			err := httpListen(h, listenIP, listenHTTPPort, nil)
			if err != nil {
				log.Fatalf("http error %s:%d: %s", listenIP, listenHTTPPort, err)
			}
		}()
	}

	// maybe later we can be smarter; now we just wait forever or until something
	// "fatals" out
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func localNet(ip net.IP) bool {
	for _, n := range localNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
