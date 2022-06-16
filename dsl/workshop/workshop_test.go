package workshop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"
)

func TestOpenWorkshop(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	workshop, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	// utils.Dump(workshop)
	assert.Equal(t, 11, len(workshop.Require))
	assert.Equal(t, 34, len(workshop.Mapping))

	// utils.Dump(workshop)

	assert.Equal(t, true, workshop.Require[3].Replaced)
	assert.Equal(t, false, workshop.Require[3].Downloaded)
	assert.Equal(t, "github.com/yaoapp/demo-wms/cloud@e86eab4c8490", workshop.Require[3].URL)
	assert.Equal(t, "github.com/yaoapp/demo-wms", workshop.Require[3].Addr)
	assert.Equal(t, "github.com", workshop.Require[3].Domain)
	assert.Equal(t, "yaoapp", workshop.Require[3].Owner)
	assert.Equal(t, "demo-wms", workshop.Require[3].Repo)
	assert.Equal(t, "/cloud", workshop.Require[3].Path)
	assert.Equal(t, "github.com/demo-wms/yaoapp/cloud", workshop.Require[3].Name)
	assert.Equal(t, "github.com/demo-wms/yaoapp/cloud", workshop.Require[3].Alias)
	assert.Equal(t, "0.0.0-e86eab4c8490", workshop.Require[3].Version.String())
	assert.Equal(t, "e86eab4c8490", workshop.Require[3].Rel)

}

func TestWorkshopGetBlank(t *testing.T) {
	root := tempAppRoot()
	workshop, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(workshop.Require))
	get(t, workshop, "github.com/yaoapp/demo-wms/cloud", "wms")
	assert.Equal(t, 1, len(workshop.Require))
	assert.Equal(t, 4, len(workshop.Mapping))
	assert.Equal(t, false, workshop.Require[0].Replaced)
	assert.Equal(t, true, workshop.Require[0].Downloaded)
	assert.Equal(t, "github.com/yaoapp/demo-wms/cloud@0.9.5", workshop.Require[0].URL)
	assert.Equal(t, "github.com/yaoapp/demo-wms", workshop.Require[0].Addr)
	assert.Equal(t, "github.com", workshop.Require[0].Domain)
	assert.Equal(t, "yaoapp", workshop.Require[0].Owner)
	assert.Equal(t, "demo-wms", workshop.Require[0].Repo)
	assert.Equal(t, "/cloud", workshop.Require[0].Path)
	assert.Equal(t, "github.com/demo-wms/yaoapp/cloud", workshop.Require[0].Name)
	assert.Equal(t, "wms", workshop.Require[0].Alias)
	assert.Equal(t, "0.9.5", workshop.Require[0].Version.String())
	assert.Equal(t, "0.9.5", workshop.Require[0].Rel)
}

func TestWorkshopGetBlankDeep(t *testing.T) {
	root := tempAppRoot()
	workshop, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(workshop.Require))
	get(t, workshop, "github.com/yaoapp/workshop-tests-wms", "wms")
	assert.Equal(t, len(workshop.Require), 4)
	indirect := 0
	for _, pkg := range workshop.Require {
		if pkg.Indirect {
			indirect++
		}
	}
	assert.Equal(t, indirect, 3)
}

func TestWorkshopSaveBlank(t *testing.T) {
	root := tempAppRoot()
	workshop, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	get(t, workshop, "github.com/yaoapp/workshop-tests-wms@04b2b1b", "wms")
	err = workshop.Save()
	if err != nil {
		t.Fatal(err)
	}

	assert.FileExists(t, workshop.file)
	content, err := os.ReadFile(workshop.file)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 335, len(content))

}

func TestWorkshopRemoveBlankDeep(t *testing.T) {
	root := tempAppRoot()
	workshop, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(workshop.Require))
	get(t, workshop, "github.com/yaoapp/workshop-tests-wms@04b2b1b", "wms")
	get(t, workshop, "github.com/yaoapp/workshop-tests-erp@v0.1.0", "erp")
	assert.Equal(t, len(workshop.Require), 4)

	workshop.Remove("github.com/yaoapp/workshop-tests-wms@04b2b1b")
	assert.FileExists(t, workshop.file)
	assert.Equal(t, len(workshop.Require), 1)
}

func get(t *testing.T, workshop *Workshop, url string, alias string) {
	cnt := 0
	err := workshop.Get(url, alias, func(total uint64, pkg *Package, status string) {
		if status == "prepare" && cnt != 0 {
			fmt.Printf("\n")
			return
		}
		fmt.Printf("\r%s", strings.Repeat(" ", 80))
		size := ""
		message := "Cached"
		if status == "downloading" {
			size = humanize.Bytes(total)
			message = "Completed"
		}

		fmt.Printf("\rGET %s... %s %s", pkg.Unique, size, message)
		cnt++
	})
	fmt.Printf("\n")
	if err != nil {
		t.Fatal(err)
	}
}

func tempAppRoot() string {
	prefix := time.Now().Format("20060102150405000000")
	root := filepath.Join(os.TempDir(), prefix)
	os.RemoveAll(root)
	os.MkdirAll(root, 0755)
	return root
}
