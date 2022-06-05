package repo

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/log"
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

// Unzip the repository
func (repo *Repo) Unzip(zipfile, dest string) error {

	// Validate dest path
	_, err := os.Stat(dest)
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s exists", dest)
	}

	// Read zip file
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}

	defer func() {
		if err := r.Close(); err != nil {
			log.Error("repo unzip: %s", err.Error())
		}
	}()

	// Unzip to temp dir
	tmpdir, err := tempFile(12, "unzip")
	if err != nil {
		return err
	}
	os.MkdirAll(tmpdir, 0755)

	path := ""
	for i, f := range r.File {
		if i == 0 {
			path = filepath.Join(tmpdir, strings.TrimRight(f.Name, "/"))
		}
		err := extractFile(f, tmpdir)
		if err != nil {
			return err
		}
	}

	// Move to dest
	dir := filepath.Dir(dest)
	_, err = os.Stat(dir)
	if !errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Rename directory
	err = os.Rename(path, dest)
	if err != nil {
		return err
	}

	return nil
}

// Latest get the repository latest version
func (repo *Repo) Latest() (string, error) {

	// tags
	tags, err := repo.Call.Tags(1, 1)
	if err != nil {
		return "", err
	}
	if len(tags) == 1 {
		return tags[0], nil
	}

	// commits
	commits, err := repo.Call.Commits(1, 1)
	if err != nil {
		return "", err
	}

	if len(commits) < 1 {
		return "", fmt.Errorf("commits not found")
	}

	return commits[0], nil
}

// extractFile extract and save file to the dest path
func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			log.Error("repo unzip extractFile: %s", err.Error())
		}
	}()

	path := filepath.Join(dest, f.Name)

	// Check for ZipSlip (Directory traversal)
	if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", path)
	}

	if f.FileInfo().IsDir() {
		os.MkdirAll(path, f.Mode())
	} else {
		os.MkdirAll(filepath.Dir(path), f.Mode())
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Error("repo unzip extractFile: %s", err.Error())
			}
		}()
		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}
	return nil
}
