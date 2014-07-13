package main

import (
	"bytes"
	"log"
	"net/http"
	"time"
)

var client *http.Client

const posterCount = 10

func init() {
	tr := &http.Transport{
		ResponseHeaderTimeout: time.Second * 5,
	}

	client = &http.Client{Transport: tr}
}

func reportPoster(ch logChannel) {
	var active bool

	url := "http://" + *flagreporthost + "/api/v1/store-result"

	if len(*flagreporthost) > 0 {
		active = true
	}

	var js []byte
	var err error

	for {
		select {
		case data := <-ch:
			log.Printf("got log data: '%#v'", data)

			if !active {
				log.Println("report poster not active")
				continue
			}

			js, err = data.JSON()
			if err != nil {
				log.Println("Could not encode JSON: ", err)
				continue
			}
			reader := bytes.NewReader(js)
			req, err := http.NewRequest("POST", url, reader)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error posting data: %s", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				log.Printf("Unhappy response: %d\n", resp.StatusCode)
				time.Sleep(200 * time.Millisecond) // Slow down a tiny bit when we have errors
			}
		}
	}
}
