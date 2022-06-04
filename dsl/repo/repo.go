package repo

import (
	"fmt"
	"strings"
)

// NewRepo create a new repo
func NewRepo(url string, option map[string]interface{}) (*Repo, error) {

	uri := strings.Split(strings.ToLower(url), "/")
	if len(uri) < 3 {
		return nil, fmt.Errorf("the given url is not correct. url: %s", url)
	}

	var api API
	if uri[0] == "github.com" {
		github := &Github{
			Owner: uri[1],
			Repo:  uri[2],
		}
		if token, has := option["token"].(string); has {
			github.Token = token
		}
		api = github
	} else {
		api = &Git{}
	}

	return &Repo{
		Domain: uri[0],
		Owner:  uri[1],
		Repo:   uri[2],
		Call:   api,
	}, nil
}

// Content Get file Content
func (repo *Repo) Content(file string) ([]byte, error) {
	return repo.Call.Content(file)
}

// Dir Get dirs
func (repo *Repo) Dir(file string) ([]string, error) {
	return repo.Call.Dir(file)
}

// Download a repository archive (zip)
func (repo *Repo) Download(rel string, process func(total uint64)) (string, error) {
	return repo.Call.Download(rel, process)
}

// Clone into local
func (repo *Repo) Clone() {}
