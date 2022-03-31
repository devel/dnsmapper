package main

import (
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

func setupSOA() *dns.SOA {

	s := *flagdomain + ". 3600 IN SOA " +
		primaryNsList[0] + " hostmaster " +
		"1 5400 5400 2419200 300"

	rr, err := dns.NewRR(s)

	if err != nil {
		log.Println("SOA Error", err)
		panic("Could not setup SOA")
	}

	return rr.(*dns.SOA)
}

func setupNS() []dns.RR {

	var nsList []dns.RR

	for _, ns := range primaryNsList {
		s := *flagdomain + ". 20800 IN NS " + ns + "."

		rr, err := dns.NewRR(s)

		if err != nil {
			log.Println("NS Error", err)
			panic("Could not setup NS")
		}
		nsList = append(nsList, rr)
	}

	return nsList
}

func getEdnsSubNet(req *dns.Msg) (enum string, rr *dns.OPT, edns *dns.EDNS0_SUBNET) {

	for _, extra := range req.Extra {
		// log.Println("Extra:", extra)
		for _, o := range extra.(*dns.OPT).Option {
			// opt_rr = extra.(*dns.OPT)
			switch e := o.(type) {
			case *dns.EDNS0_NSID:
				// do stuff with e.Nsid
			case *dns.EDNS0_SUBNET:
				// log.Println("Got edns", e.Address, e.Family, e.SourceNetmask, e.SourceScope)
				if e.Address != nil {
					edns = e
					rr = extra.(*dns.OPT)
					enum = fmt.Sprintf("%s/%d", e.Address.String(), e.SourceNetmask)
				}
			}
		}
	}
	return
}

func setupServerFunc() func(dns.ResponseWriter, *dns.Msg) {

	soa := setupSOA()
	ns := setupNS()

	h := &dns.RR_Header{Ttl: 5, Class: dns.ClassINET, Rrtype: dns.TypeA}
	a := &dns.A{Hdr: *h, A: net.ParseIP(*flagip)}

	hasACME := false
	if len(*flagacmedomain) > 0 {
		hasACME = true
		if !dns.IsFqdn(*flagacmedomain) {
			*flagacmedomain = *flagacmedomain + "."
		}
	}

	return func(w dns.ResponseWriter, req *dns.Msg) {

		m := new(dns.Msg)
		m.SetReply(req)
		if e := m.IsEdns0(); e != nil {
			m.SetEdns0(4096, e.Do())
		}
		m.Authoritative = true

		uuid := getUUIDFromDomain(req.Question[0].Name)

		qtype := req.Question[0].Qtype

		ednsIP, extraRR, edns := getEdnsSubNet(req)
		ip, _, _ := net.SplitHostPort(w.RemoteAddr().String())

		if edns != nil {
			// log.Println("family", edns.Family)
			if edns.Family != 0 {
				edns.SourceScope = 0
				m.Extra = append(m.Extra, extraRR)
			}
		}

		if qtype == dns.TypeNS && len(uuid) == 0 {
			m.Answer = ns
			w.WriteMsg(m)
			return
		}

		log.Printf("DNS request from %s for %s", ip, uuid)

		if hasACME && uuid == "_acme-challenge" {
			acmeCNAME := &dns.CNAME{
				Hdr: dns.RR_Header{
					Name:   req.Question[0].Name,
					Ttl:    600,
					Rrtype: dns.TypeCNAME, Class: dns.ClassINET,
				},
				Target: *flagacmedomain,
			}
			m.Answer = []dns.RR{acmeCNAME}
			w.WriteMsg(m)
			return
		}

		// we only know how to do A records
		if qtype != dns.TypeA {
			m.Ns = []dns.RR{soa}
			w.WriteMsg(m)
			return
		}

		if len(uuid) == 0 {
			// NOERROR
			m.Ns = []dns.RR{soa}
			w.WriteMsg(m)
			return
		}

		if len(m.Answer) == 0 {
			a.Header().Name = req.Question[0].Name
			m.Answer = []dns.RR{a}

			if uuid == "www" {
				// we always redirect on 'www' so tell DNS caches
				// it is good for a little longer and don't store
				// the session
				a.Header().Ttl = 120
			} else {
				// We expire the session data after 10 seconds, so
				// encourage DNS caches to come back after 5.
				a.Header().Ttl = 5
				setCache(uuid, ip, ednsIP)
				if edns != nil {
					edns.SourceScope = edns.SourceNetmask
				}
			}
		}

		if len(m.Answer) == 0 {
			// return NXDOMAIN
			m.SetRcode(req, dns.RcodeNameError)
			m.Authoritative = true
			m.Ns = []dns.RR{soa}
		}

		// log.Println("Returning", m)

		w.WriteMsg(m)
	}

}

func listenAndServeDNS(ip string, port int) {

	listen := fmt.Sprintf("%s:%d", ip, port)

	prots := []string{"udp", "tcp"}

	for _, prot := range prots {
		go func(p string) {
			server := &dns.Server{Addr: listen, Net: p}

			log.Printf("DNS listen on %s %s", ip, p)
			if err := server.ListenAndServe(); err != nil {
				log.Fatalf("geodns: failed to setup dns %s %s: %s", ip, p, err)
			}
			log.Fatalf("geodns: ListenAndServe unexpectedly returned")
		}(prot)
	}

}
