package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	. "gopkg.in/check.v1"
)

type HttpSuite struct {
	srv *httptest.Server
	mux *http.ServeMux
	uri *url.URL
}

var _ = Suite(&HttpSuite{})

func (s *HttpSuite) SetUpSuite(c *C) {
	s.mux = buildMux()
	s.srv = httptest.NewServer(s.mux)
	s.uri, _ = url.Parse(s.srv.URL)

}

func (s *HttpSuite) TearDownSuite(c *C) {
	s.srv.Close()
}

type jsResponse struct {
	http *http.Response
	js   map[string]interface{}
}

func (s *HttpSuite) request(c *C, path, data string) (*jsResponse, error) {
	uri := *s.uri
	uri.Path = path

	c.Logf("Posting to %s: %#v", uri.String(), data)

	// resp, err := http.PostForm(uri.String(), data)
	resp, err := http.Post(uri.String(), "application/json", bytes.NewBufferString(data))
	if err != nil {
		c.Fatalf("Could not post %s: %s", uri.String(), err)
	}
	defer resp.Body.Close()

	c.Assert(resp.StatusCode, Equals, 200)

	if p, err := ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	} else {

		c.Logf("response data: '%s'", p)

		r := new(jsResponse)

		r.http = resp

		err = json.Unmarshal(p, &r.js)
		return r, err
	}
}

func (s *HttpSuite) TestPost(c *C) {

	resp, err := s.request(c,
		"/api/v1/store-result", `{
			"ServerIP": "207.171.3.4",
			"ClientIP": "140.211.11.133"
		}`,
	)
	c.Assert(err, IsNil)
	c.Check(resp.js["ServerIP"].(string), Equals, "207.171.3.4")
	fmt.Printf("resp: %#v\n", resp)

}
