package dsl

import (
	"strings"

	"github.com/yaoapp/kun/utils"
)

// compile compile the content
func (yao *YAO) compile() error {

	err := yao.compileFrom()
	if err != nil {
		return err
	}

	return nil
}

// compileFrom from
func (yao *YAO) compileFrom() error {
	if yao.Head.From == "" {
		return nil
	}

	if strings.HasPrefix(yao.Head.From, "@") {
		return yao.compileFromRemote()
	}

	return yao.compileFromLocal()
}

func (yao *YAO) compileFromRemote() error {
	utils.Dump(yao)
	return nil
}

func (yao *YAO) compileFromRemoteAlias() {}

func (yao *YAO) compileFromLocal() error { return nil }
