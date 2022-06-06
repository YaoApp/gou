package repo

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/dns"
	"github.com/yaoapp/gou/network"
)

// Github API
type Github struct {
	Owner string
	Repo  string
	Token string
}

// Content Get file Content via Github API
func (github *Github) Content(file string) ([]byte, error) {

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents%s",
		github.Owner, github.Repo, github.path(file),
	)

	resp := network.RequestGet(url, nil, github.headers(nil))
	if resp.Status != 200 {
		return nil, fmt.Errorf("Github API Error: %d %s", resp.Status, github.error(resp))
	}

	data := github.data(resp)
	content, ok := data["content"].(string)
	if !ok {
		return nil, fmt.Errorf("Github API contents Error: %s", resp.Body)
	}
	bytes, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// Dir Get folders via Github API
func (github *Github) Dir(path string) ([]string, error) {

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents%s",
		github.Owner, github.Repo, github.path(path),
	)

	resp := network.RequestGet(url, nil, github.headers(nil))
	if resp.Status != 200 {
		return nil, fmt.Errorf("Github API dir Error: %d %s", resp.Status, github.error(resp))
	}

	res := []string{}
	data := github.arrayData(resp)
	for _, row := range data {
		if path, has := row["path"].(string); has {
			res = append(res, github.path(path))
		}
	}
	return res, nil
}

// Tags get the tags of the repository via Github API
func (github *Github) Tags(page, perpage int) ([]string, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/tags?per_page=%d&page=%d",
		github.Owner, github.Repo, perpage, page,
	)

	resp := network.RequestGet(url, nil, github.headers(nil))
	if resp.Status != 200 {
		return nil, fmt.Errorf("Github API tags Error: %d %s", resp.Status, github.error(resp))
	}

	res := []string{}
	data := github.arrayData(resp)
	for _, row := range data {
		if name, has := row["name"].(string); has {
			res = append(res, name)
		}
	}
	return res, nil
}

// Commits get the commits of the repository via Github API
func (github *Github) Commits(page, perpage int) ([]string, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/commits?per_page=%d&page=%d",
		github.Owner, github.Repo, perpage, page,
	)

	resp := network.RequestGet(url, nil, github.headers(nil))
	if resp.Status != 200 {
		return nil, fmt.Errorf("Github API commits Error: %d %s", resp.Status, github.error(resp))
	}

	res := []string{}
	data := github.arrayData(resp)
	for _, row := range data {
		if sha, has := row["sha"].(string); has {
			if len(sha) > 12 {
				res = append(res, sha[0:12])
			}
		}
	}
	return res, nil
}

// Download a repository archive (zip) via Github API
// Docs: https://docs.github.com/en/rest/repos/contents#download-a-repository-archive-zip
func (github *Github) Download(rel string, process func(total uint64)) (string, error) {

	p := &downloadProcess{call: process}
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/zipball/%s",
		github.Owner, github.Repo, rel,
	)

	// Create a temp file
	tmpfile, err := tempFile(12, "zip")
	if err != nil {
		return "", err
	}

	tmp, err := os.Create(tmpfile)
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	// Create a get request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set request headers
	for name, header := range github.headers(nil) {
		req.Header.Set(name, header)
	}

	// Force using system DSN resolver
	// var dialer = &net.Dialer{Resolver: &net.Resolver{PreferGo: false}}
	var dialContext = dns.DialContext()
	var client *http.Client = &http.Client{Transport: &http.Transport{DialContext: dialContext}}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s", resp.Status)
	}

	// Copy to the temp file
	if _, err = io.Copy(tmp, io.TeeReader(resp.Body, p)); err != nil {
		return "", err
	}

	return tmpfile, nil
}

func (github *Github) error(resp network.Response) string {
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return resp.Body
	}

	if message, ok := data["message"].(string); ok {
		return message
	}
	return resp.Body
}

func (github *Github) data(resp network.Response) map[string]interface{} {
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return data
}

func (github *Github) arrayData(resp network.Response) []map[string]interface{} {
	res := []map[string]interface{}{}
	array, ok := resp.Data.([]interface{})
	if !ok {
		return res
	}
	for _, arr := range array {
		if row, ok := arr.(map[string]interface{}); ok {
			res = append(res, row)
		}
	}
	return res
}

func (github *Github) headers(headers map[string]string) map[string]string {

	if headers == nil {
		headers = map[string]string{}
	}

	if github.Token != "" {
		headers["authorization"] = fmt.Sprintf("token %s", github.Token)
	}

	headers["Accept"] = "application/vnd.github.v3+json"
	return headers
}

func (github *Github) path(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = filepath.Join("/", path)
	}
	return path
}
