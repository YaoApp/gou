package dns

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestLookupIP(t *testing.T) {
	_, err := LookupIP("localhost", true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDialContext(t *testing.T) {
	var body []byte
	req, err := http.NewRequest("GET", "https://api.github.com/users/yaoapp", bytes.NewBuffer(body))
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext:     DialContext(),
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		t.Fatal(err)
	}
}
