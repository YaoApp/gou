package repo

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

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
		return nil, fmt.Errorf("Github API Error: %s", resp.Body)
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
		return nil, fmt.Errorf("Github API Error: %d %s", resp.Status, github.error(resp))
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
