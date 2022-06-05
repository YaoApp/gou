package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"
)

func TestRepoUnzip(t *testing.T) {

	repo, err := NewRepo("github.com/yaoapp/demo-crm", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	tmpfile, err := repo.Download("3bcaf04ae28f", func(total uint64) {
		fmt.Printf("\r%s", strings.Repeat(" ", 35))
		fmt.Printf("\rDownloading... %s Complete", humanize.Bytes(total))
	})

	fmt.Println("")
	if err != nil {
		t.Fatal(err)
	}
	assert.FileExists(t, tmpfile)

	dest, err := tempFile(12, "unzip")
	if err != nil {
		t.Fatal(err)
	}

	err = repo.Unzip(tmpfile, dest)
	if err != nil {
		t.Fatal(err)
	}

	assert.DirExists(t, dest)
	assert.FileExists(t, filepath.Join(dest, "README.md"))
	os.Remove(dest)
	os.Remove(tmpfile)
}
