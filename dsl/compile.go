package dsl

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/dsl/workshop"
	"github.com/yaoapp/kun/log"
)

// compile compile the content
func (yao *YAO) compile() error {

	// COPY Content
	yao.Compiled = yao.Content

	// Compile From
	err := yao.compileFrom()
	if err != nil {
		return err
	}

	return nil
}

// compileFrom FROM
func (yao *YAO) compileFrom() error {
	if yao.Head.From == "" {
		return nil
	}

	if strings.HasPrefix(yao.Head.From, "@") {
		return yao.compileFromRemote()
	}

	return yao.compileFromLocal()
}

// compileFromRemote FROM the remote package
func (yao *YAO) compileFromRemote() error {

	remoteWorkshop, file, err := yao.fromRemoteFile()
	if err != nil {
		return err
	}

	// Trace
	yao.Trace = append(yao.Trace, file)

	remote := New(remoteWorkshop)
	err = remote.Open(file)
	if err != nil {
		return err
	}

	// Compile remote file
	err = remote.Compile()
	if err != nil {
		return err
	}

	// Append the remote trace
	yao.Trace = append(yao.Trace, remote.Trace...)

	// Merge Remote Content
	yao.merge(remote.Compiled)

	return nil
}

// merge mege the content
func (yao *YAO) merge(content map[string]interface{}) error {
	yao.Compiled = content
	return nil
}

// fromPath get the remote file
func (yao *YAO) fromRemoteFile() (remoteWorkshop *workshop.Workshop, file string, err error) {

	// VALIDATE THE FROM
	if yao.Head.From == "" {
		return nil, "", fmt.Errorf("FROM is null")
	}

	if yao.Head.From[0] != '@' {
		return nil, "", fmt.Errorf("FROM is not remote")
	}

	// COMPUTE THE PACKAGE NAME
	from := yao.Head.From[1:]
	fromArr := strings.Split(from, "/")
	if len(fromArr) < 4 {
		return nil, "", fmt.Errorf("FROM is error %s", from)
	}
	name := strings.Join(fromArr[:3], "/")

	// AUTO GET PACKAGE
	if !yao.Workshop.Has(name) {
		err = yao.Workshop.Get(name, "", func(total uint64, pkg *workshop.Package, message string) {
			log.Trace("GET %s %d ... %s", pkg.Unique, total, message)
		})
		if err != nil {
			return nil, "", fmt.Errorf("The package %s does not loaded. %s", name, err.Error())
		}
	}

	// AUTO DOWNLOAD
	isDownload, err := yao.Workshop.Mapping[name].IsDownload()
	if err != nil {
		return nil, "", fmt.Errorf("download the package %s error. %s", name, err.Error())
	}

	if !isDownload {
		_, err = yao.Workshop.Download(yao.Workshop.Mapping[name], func(total uint64, pkg *workshop.Package, message string) {
			log.Trace("Download %s %d ... %s", pkg.Unique, total, message)
		})
		if err != nil {
			return nil, "", fmt.Errorf("download the package %s error. %s", name, err.Error())
		}
	}

	// OPEN REMOTE WORKSHOP
	remoteWorkshop, err = workshop.Open(yao.Workshop.Mapping[name].LocalPath)
	if err != nil {
		return nil, "", err
	}

	// Extra file
	pathArr := []string{yao.Workshop.Mapping[name].LocalPath}
	pathArr = append(pathArr, fromArr[3:]...)
	file = filepath.Join(pathArr...) + fmt.Sprintf(".%s.yao", TypeExtensions[yao.Head.Type])

	return remoteWorkshop, file, nil
}

func (yao *YAO) compileFromRemoteAlias() {}

func (yao *YAO) compileFromLocal() error { return nil }
