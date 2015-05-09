package main

import (
	"fmt"
	"log"
	"time"
)

// Session is what we store temporarily from each DNS requestxxrf
type Session struct {
	IP     string
	EDNS   string
	Expire int64
}

func setCache(uuid, ip, ednsIP string) error {

	session := &Session{
		Expire: time.Now().Add(10 * time.Second).Unix(),
		IP:     ip,
		EDNS:   ednsIP,
	}

	ok := cache.Add("dns-"+uuid, session)
	if !ok {
		return fmt.Errorf("%s not saved to the cache", uuid)
	}
	return nil
}

func getCache(uuid string) (string, string, bool) {
	// Use Peek instead of get to just have a "fifo" cache,
	// where adding an item again moves it to the front of the
	// list again.
	get, ok := cache.Peek("dns-" + uuid)
	if !ok {
		return "", "", false
	}

	s, ok := get.(*Session)
	if !ok {
		log.Printf("Session %s wasn't a session type (%T)", uuid, get)
		return "", "", false
	}

	if s.Expire < time.Now().Unix() {
		return "", "", false
	}

	return s.IP, s.EDNS, true
}
