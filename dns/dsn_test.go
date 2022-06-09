package dns

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupIP(t *testing.T) {
	_, err := LookupIP("localhost", true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLookupIPCaches(t *testing.T) {
	ips, err := LookupIP("localhost", true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ips, caches["localhost_true"])

	_, has := caches["localhost_false"]
	assert.Equal(t, false, has)

	ips, err = LookupIP("localhost", false)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, ips, caches["localhost_false"])

}

func TestLookupIPHostIsIP(t *testing.T) {
	ips, err := LookupIP("127.0.0.1", true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, ips, []string{"127.0.0.1"})
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
