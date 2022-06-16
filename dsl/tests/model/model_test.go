package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/gou/dsl"
	"github.com/yaoapp/gou/dsl/workshop"
)

func TestDSLCompileSimple(t *testing.T) {
	newModel(t, "simple.mod.yao")
}

func TestDSLCheck(t *testing.T) {}

func TestDSLRefresh(t *testing.T) {}

func TestDSLRegister(t *testing.T) {}

func TestDSLChange(t *testing.T) {}

func newModel(t *testing.T, name string) *dsl.YAO {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := filepath.Join(root, "models", name)
	workshop, err := workshop.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	yao := dsl.New(workshop)
	err = yao.Open(file)
	if err != nil {
		t.Fatal(err)
	}

	err = yao.Compile()
	if err != nil {
		t.Fatal(err)
	}
	return yao
}
