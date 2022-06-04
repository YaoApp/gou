package dsl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
	jsoniter "github.com/json-iterator/go"
)

// UnmarshalJSON for json
func (pkg *Package) UnmarshalJSON(data []byte) error {

	if len(data) < 1 {
		return fmt.Errorf("package should be {\"key\":\"value\"} or \"value\" format, but got nothing")
	}

	if data[0] == '{' { // map format: {"key":"value"}
		input := map[string]string{}
		err := jsoniter.Unmarshal(data, &input)
		if err != nil {
			return fmt.Errorf("package should be {\"key\":\"value\"} or \"value\" format, but got: %s", data)
		}

		for alias, url := range input {
			if err := pkg.Set(url, alias); err != nil {
				return err
			}
			break
		}
		return nil

	} else if data[0] == '"' { // string format: "value"

		if err := pkg.Set(string(data[1:len(data)-1]), ""); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("should be {\"key\":\"value\"} or \"value\" format, but got: %s", data)
}

// MarshalJSON for json
func (pkg Package) MarshalJSON() ([]byte, error) {
	if pkg.Name == pkg.Alias {
		return jsoniter.Marshal(pkg.URL)
	}
	uri := map[string]string{}
	uri[pkg.Alias] = pkg.URL
	return jsoniter.Marshal(uri)
}

// Map package to map[string]interface{}
func (pkg Package) Map() map[string]interface{} {
	return map[string]interface{}{
		"url":        pkg.URL,
		"addr":       pkg.Addr,
		"name":       pkg.Name,
		"alias":      pkg.Alias,
		"domain":     pkg.Domain,
		"owner":      pkg.Owner,
		"repo":       pkg.Repo,
		"path":       pkg.Path,
		"version":    pkg.Version.String(),
		"commit":     pkg.Commit,
		"localpath":  pkg.LocalPath,
		"downloaded": pkg.Downloaded,
		"replaced":   pkg.Replaced,
	}
}

// Set set repo and alias
func (pkg *Package) Set(url string, alias string) error {
	uri := strings.Split(url, "@")
	if len(uri) != 2 {
		return fmt.Errorf("package url should be \"repo@version\" format, but got: %s", url)
	}

	err := pkg.SetAddr(uri[0])
	if err != nil {
		return err
	}

	err = pkg.SetVersion(uri[1])
	if err != nil {
		return err
	}

	err = pkg.SetLocalPath()
	if err != nil {
		return err
	}

	pkg.URL = url
	pkg.Alias = alias
	if alias == "" {
		pkg.Alias = pkg.Name
	}

	return nil
}

// SetAddr parse and set repo, domain, owner, repo, path and name
func (pkg *Package) SetAddr(url string) error {
	url = strings.ToLower(url)
	uri := strings.Split(url, "/")
	if len(uri) < 3 {
		return fmt.Errorf("package url should be a git repo. \"domain/org/repo/path\", but got: %s", url)
	}

	pkg.Domain = uri[0]
	pkg.Owner = uri[1]
	pkg.Repo = uri[2]
	pkg.Path = "/"
	name := fmt.Sprintf("%s.%s", pkg.Repo, pkg.Owner)
	if len(uri) > 3 {
		pkg.Path = fmt.Sprintf("/%s", filepath.Join(uri[3:]...))
		name = fmt.Sprintf("%s.%s", name, strings.Join(uri[3:], "."))
	}
	pkg.Name = name
	pkg.Addr = fmt.Sprintf("%s/%s/%s", pkg.Domain, pkg.Owner, pkg.Repo)
	return nil
}

// SetVersion parse and set version, commit
func (pkg *Package) SetVersion(ver string) error {
	ver = strings.TrimLeft(strings.ToLower(ver), "v")
	version, err := semver.New(ver)
	if err != nil {
		return fmt.Errorf("package version should be Semantic Versioning 2.0.0 format, but got: %s, error: %s", ver, err)
	}
	pkg.Version = *version
	if pkg.Version.Pre != nil && pkg.Version.Pre[0].VersionStr != "" {
		vstr := strings.Split(pkg.Version.Pre[0].VersionStr, "-")
		pkg.Commit = vstr[len(vstr)-1]
	}
	return nil
}

// SetLocalPath get the local root path
func (pkg *Package) SetLocalPath() error {
	root, err := WorkshopRoot()
	if err != nil {
		return err
	}
	paths := strings.Split(pkg.Path, "/")
	version := pkg.Commit
	if version == "" {
		version = pkg.Version.String()
	}
	pkg.LocalPath = filepath.Join(
		root,
		pkg.Domain, pkg.Owner,
		fmt.Sprintf("%s@%s", pkg.Repo, version),
		filepath.Join(paths...),
	)
	return nil
}

// IsDownload check if the package has been downloaded.
func (pkg *Package) IsDownload() (bool, error) {
	pkg.Downloaded = false
	_, err := os.Stat(pkg.LocalPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	pkg.Downloaded = true
	return true, nil
}

// FileContent get the repo file content
func (pkg *Package) FileContent(file string) {
}
