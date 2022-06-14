package workshop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/dsl/repo"
	"github.com/yaoapp/gou/dsl/u"
)

// UnmarshalJSON for json
func (pkg *Package) UnmarshalJSON(data []byte) error {

	if len(data) < 1 {
		return fmt.Errorf("package should be {\"key\":\"value\"} or \"value\" format, but got nothing")
	}

	if data[0] == '{' { // map format: {"key":"value"}
		input := map[string]interface{}{}
		err := jsoniter.Unmarshal(data, &input)
		if err != nil {
			return fmt.Errorf("package should be {\"key\":\"value\"} or \"value\" format, but got: %s", data)
		}

		// Set indirect
		if indirect, has := input["indirect"].(bool); has {
			pkg.SetIndirect(indirect)
		}

		if url, has := input["repo"].(string); has {
			alias := ""
			if as, ok := input["alias"].(string); ok {
				alias = as
			}

			if err := pkg.Set(url, alias); err != nil {
				return err
			}
			return nil

		}

		for alias, value := range input {
			url, ok := value.(string)
			if !ok {
				return fmt.Errorf("package should be {\"key\":\"value\"} or \"value\" format, but got: %s", data)
			}

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

	if pkg.Indirect {
		uri := map[string]interface{}{}
		uri["indirect"] = pkg.Indirect
		if pkg.Name == pkg.Alias {
			uri["repo"] = pkg.URL
		} else {
			uri[pkg.Alias] = pkg.URL
		}
		return jsoniter.Marshal(uri)
	}

	if pkg.Name == pkg.Alias {
		return jsoniter.Marshal(pkg.URL)
	}

	uri := map[string]interface{}{}
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
		"rel":        pkg.Rel,
		"localpath":  pkg.LocalPath,
		"downloaded": pkg.Downloaded,
		"replaced":   pkg.Replaced,
		"unique":     pkg.Unique,
		"indirect":   pkg.Indirect,
	}
}

// Set set repo and alias
func (pkg *Package) Set(url string, alias string) error {
	uri := strings.Split(url, "@")
	if len(uri) != 2 {
		return fmt.Errorf("package url should be \"repo@version\" format, but got: %s", url)
	}

	err := pkg.SetVersion(uri[1])
	if err != nil {
		return err
	}

	err = pkg.SetAddr(uri[0])
	if err != nil {
		return err
	}

	err = pkg.SetLocalPath()
	if err != nil {
		return err
	}

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
	name := fmt.Sprintf("%s/%s/%s", pkg.Domain, pkg.Repo, pkg.Owner)
	if len(uri) > 3 {
		pkg.Path = fmt.Sprintf("/%s", filepath.Join(uri[3:]...))
		name = fmt.Sprintf("%s/%s", name, strings.Join(uri[3:], "/"))
	}
	pkg.Name = name
	pkg.Addr = fmt.Sprintf("%s/%s/%s", pkg.Domain, pkg.Owner, pkg.Repo)

	// Set URL
	path := pkg.Path
	if path == "/" {
		path = ""
	}
	pkg.URL = fmt.Sprintf("%s%s@%s", pkg.Addr, path, pkg.Rel)
	pkg.Unique = fmt.Sprintf("%s@%s", pkg.Addr, pkg.Rel)
	return nil
}

// SetVersion parse and set version, commit
func (pkg *Package) SetVersion(ver string) error {

	version, err := semver.New(strings.TrimLeft(strings.ToLower(ver), "v"))
	if err != nil {

		if len(ver) <= 32 { //Commint
			pkg.Rel = ver
			version, _ = semver.New(fmt.Sprintf("0.0.0-%s", ver))
			pkg.Version = *version
			pkg.Rel = ver
			return nil
		}

		return fmt.Errorf("package version should be Semantic Versioning 2.0.0 format, but got: %s, error: %s", ver, err)
	}

	pkg.Version = *version
	pkg.Rel = ver
	if pkg.Version.Pre != nil && pkg.Version.Pre[0].VersionStr != "" {
		vstr := strings.Split(pkg.Version.Pre[0].VersionStr, "-")
		pkg.Rel = vstr[len(vstr)-1]
	}

	return nil
}

// SetLocalPath get the local root path
func (pkg *Package) SetLocalPath() error {
	root, err := Root()
	if err != nil {
		return err
	}
	paths := strings.Split(pkg.Path, "/")
	pkg.LocalPath = filepath.Join(
		root,
		pkg.Domain, pkg.Owner,
		fmt.Sprintf("%s@%s", pkg.Repo, pkg.Rel),
		filepath.Join(paths...),
	)
	return nil
}

// SetIndirect set the indirect value
func (pkg *Package) SetIndirect(indirect bool) {
	pkg.Indirect = indirect
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

// Cache get the package cache name
func (pkg *Package) Cache(root string) string {
	return filepath.Join(root, pkg.Domain, pkg.Owner, pkg.Repo, fmt.Sprintf("@%s.zip", pkg.Rel))
}

// LocalRepo get the package repo local path
func (pkg *Package) LocalRepo(root string) string {
	return filepath.Join(root, pkg.Unique)
}

// Option get the package option
func (pkg *Package) Option(cfg Config) map[string]interface{} {
	if option, has := cfg[pkg.Domain]; has {
		return option
	}
	return map[string]interface{}{}
}

// Download download the package
func (pkg *Package) Download(root string, option map[string]interface{}, process func(total uint64, pkg *Package, message string)) (string, error) {

	if process != nil {
		process(100, pkg, "prepare")
	}

	if option == nil {
		option = map[string]interface{}{}
	}

	// download from cache
	if cache, has := option["cache"].(string); has {
		cache := pkg.Cache(cache)
		exists, _ := u.FileExists(cache)
		if exists {
			if process != nil {
				process(100, pkg, "cached")
			}

			isDownload, err := pkg.IsDownload()
			if err != nil {
				return "", err
			}

			// Unzip File
			if !isDownload {
				repo, err := repo.NewRepo(pkg.Addr, option)
				if err != nil {
					return "", err
				}

				dest := pkg.LocalRepo(root)
				if exitis, _ := u.FileExists(dest); exitis {
					os.RemoveAll(dest)
				}

				err = repo.Unzip(cache, dest)
				if err != nil {
					return "", err
				}

				pkg.Downloaded = true
			}

			return cache, nil
		}
	}

	// download to cache path
	repo, err := repo.NewRepo(pkg.Addr, option)
	if err != nil {
		return "", err
	}

	var p func(total uint64) = nil
	if process != nil {
		p = func(total uint64) {
			process(total, pkg, "downloading")
		}
	}

	tmpfile, err := repo.Download(pkg.Rel, p)
	if err != nil {
		return "", err
	}

	// unzip file
	dest := pkg.LocalRepo(root)
	if exitis, _ := u.FileExists(dest); exitis {
		os.RemoveAll(dest)
	}

	err = repo.Unzip(tmpfile, dest)
	if err != nil {
		return "", err
	}

	// Mark As Downloaded
	pkg.Downloaded = true

	// cache the temp file
	if cache, has := option["cache"].(string); has {
		cache := pkg.Cache(cache)
		dir := filepath.Dir(cache)
		os.MkdirAll(dir, 0755)
		os.Rename(tmpfile, cache)
	}

	return dest, nil
}

// Dependencies get the Dependencies of the package
func (pkg *Package) Dependencies() ([]*Package, error) {

	if exists, _ := u.FileExists(filepath.Join(pkg.LocalPath, "workshop.yao")); !exists {
		return []*Package{}, nil
	}

	workshop, err := Open(pkg.LocalPath)
	if err != nil {
		return nil, err
	}
	return workshop.Require, nil
}
