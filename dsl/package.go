package dsl

import (
	"fmt"
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

		for alias, repo := range input {
			if err := pkg.Set(repo, alias); err != nil {
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
		return jsoniter.Marshal(fmt.Sprintf("%s@%s", pkg.Repo, pkg.Version.String()))
	}
	repo := map[string]string{}
	repo[pkg.Alias] = fmt.Sprintf("%s@%s", pkg.Repo, pkg.Version.String())
	return jsoniter.Marshal(repo)
}

// Set set repo and alias
func (pkg *Package) Set(url string, alias string) error {
	uri := strings.Split(url, "@")
	if len(uri) != 2 {
		return fmt.Errorf("package url should be \"repo@version\" format, but got: %s", url)
	}

	err := pkg.SetURL(uri[0])
	if err != nil {
		return err
	}

	err = pkg.SetVersion(uri[1])
	if err != nil {
		return err
	}

	pkg.Alias = alias
	if alias == "" {
		pkg.Alias = pkg.Name
	}

	return nil
}

// SetURL parse and set repo, domain, team, project, path and name
func (pkg *Package) SetURL(url string) error {
	url = strings.ToLower(url)
	uri := strings.Split(url, "/")
	if len(uri) < 3 {
		return fmt.Errorf("package url should be a git repo. \"domain/org/project/path\", but got: %s", url)
	}

	pkg.Repo = url
	pkg.Domain = uri[0]
	pkg.Team = uri[1]
	pkg.Project = uri[2]
	pkg.Path = "/"
	name := fmt.Sprintf("%s.%s", pkg.Project, pkg.Team)
	if len(uri) > 3 {
		pkg.Path = strings.Join(uri[3:], "/")
		name = fmt.Sprintf("%s.%s", name, strings.Join(uri[3:], "."))
	}
	pkg.Name = name
	return nil
}

// SetVersion parse and set version, commit
func (pkg *Package) SetVersion(ver string) error {
	ver = strings.ToLower(strings.TrimLeft(ver, "v"))
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
