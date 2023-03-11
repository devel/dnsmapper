package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devel/dnsmapper/storeapi"
	"github.com/stretchr/testify/assert"
)

type jsResponse struct {
	http *http.Response
	js   map[string]interface{}
}

func TestAPI(t *testing.T) {

	srv := httptest.NewServer(buildMux())
	uri := srv.URL + "/api/v1/myip"

	// resp, err := http.PostForm(uri.String(), data)
	resp, err := http.Get(uri)
	if err != nil {
		t.Fatalf("Could not get %s: %s", uri, err)
	}
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var p []byte
	if p, err = io.ReadAll(resp.Body); err != nil {
		t.Errorf("Error reading response: %s", err)
		return
	}

	t.Logf("response data: '%s'", p)

	ips := []storeapi.LogData{}

	err = json.Unmarshal(p, &ips)
	if err != nil {
		t.Errorf("Error parsing response: %s", err)
	}

}
