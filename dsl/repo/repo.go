package repo

import (
	"fmt"
	"strings"
)

// NewRepo create a new repo
// eg: NewRepo("github.com/yaoapp/gou", map[string]interface{}{"token": "GITHUB TOKEN"})
func NewRepo(addr string, option map[string]interface{}) (*Repo, error) {

	uri := strings.Split(strings.ToLower(addr), "/")
	if len(uri) < 3 {
		return nil, fmt.Errorf("the given addr is not correct. addr: %s", addr)
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
func (repo *Repo) Dir(path string) ([]string, error) {
	return repo.Call.Dir(path)
}

// Download the repository archive (zip)
func (repo *Repo) Download(rel string, process func(total uint64)) (string, error) {

	// @TODO
	// SOMEONE REMOVES THE REPOSITORY IT SHOULD WORK STILL
	//
	// IF THE REPOSITORY IS OPEN-SOURCE
	// 		TRY TO DOWNLOAD FROM THE REPOSITORY URL DIRECTLY
	// 		IF NOT WORKING, DOWNLOAD FROM MIRRORS.YAOAPPS.COM

	// @TIPS
	// SHOULD SHOW THE OPEN-SOURCE LICENSE OF THE REPOSITORY ON THE WORKSHOP WEBSITE

	return repo.Call.Download(rel, process)
}

// Latest get the repository latest version
func (repo *Repo) Latest() (string, error) {
	return "0.9.5", nil
}
