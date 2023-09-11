package dns

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
)

func TestLookupIP(t *testing.T) {
	_, err := LookupIP("github.com", true)
	if err != nil {
		t.Fatal(err)
	}

	res, err := LookupIP("api.mch.weixin.qq.com", true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, strings.Join(res, ","), ":")
	res, err = LookupIP("api.mch.weixin.qq.com", false)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, strings.Join(res, ","), ":")
}

func TestLookupIPCaches(t *testing.T) {
	ips, err := LookupIP("github.com", true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ips, caches["github.com_true"])
	utils.Dump(ips)

	_, has := caches["github.com_false"]
	assert.Equal(t, false, has)

	ips, err = LookupIP("github.com", false)
	if err != nil {
		t.Fatal(err)
	}
	utils.Dump(ips)
	assert.Equal(t, ips, caches["github.com_false"])

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
