package repo

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"
)

func TestGithubContentPublic(t *testing.T) {
	repo, err := NewRepo("github.com/yaoapp/gou", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	content, err := repo.Content("/tests/app/app.yao")
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(content), 0)
	assert.Contains(t, string(content), "Pet Hospital")
}

func TestGithubDir(t *testing.T) {
	repo, err := NewRepo("github.com/yaoapp/gou", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	dirs, err := repo.Dir("/tests/app")
	if err != nil {
		t.Fatal(err)
	}

	assert.Greater(t, len(dirs), 0)
	assert.Contains(t, dirs, "/tests/app/workshop.yao")
}

func TestGithubLatest(t *testing.T) {
	repo, err := NewRepo("github.com/yaoapp/gou", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := repo.Latest()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(latest), 0)
}

func TestGithubLatestCommit(t *testing.T) {
	repo, err := NewRepo("github.com/yaoapp/demo-crm", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := repo.Latest()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(latest), 12)
}

func TestGithubContentPrivate(t *testing.T) {
	url := os.Getenv("GOU_TEST_GITHUB_REPO")
	token := os.Getenv("GOU_TEST_GITHUB_TOKEN")
	repo, err := NewRepo(url, map[string]interface{}{"token": token})
	if err != nil {
		t.Fatal(err)
	}

	content, err := repo.Content("/README.md")
	if err != nil {
		t.Fatal(err)
	}

	assert.Greater(t, len(content), 0)
	assert.Contains(t, string(content), "# workshop-tests-private")
}

func TestGithubContentFail(t *testing.T) {
	repo, err := NewRepo("github.com/yaoapp/gou", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Content("/test/app/app.yao")
	assert.EqualError(t, err, "Github API Error: 404 Not Found")
}

func TestGithubDownloadPublic(t *testing.T) {

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
	os.Remove(tmpfile)
}

func TestGithubDownloadSilent(t *testing.T) {
	repo, err := NewRepo("github.com/yaoapp/demo-crm", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	tmpfile, err := repo.Download("3bcaf04ae28f", nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.FileExists(t, tmpfile)
	os.Remove(tmpfile)
}

func TestGithubDownloadPrivate(t *testing.T) {

	url := os.Getenv("GOU_TEST_GITHUB_REPO")
	token := os.Getenv("GOU_TEST_GITHUB_TOKEN")
	repo, err := NewRepo(url, map[string]interface{}{"token": token})
	if err != nil {
		t.Fatal(err)
	}

	tmpfile, err := repo.Download("main", func(total uint64) {
		fmt.Printf("\r%s", strings.Repeat(" ", 35))
		fmt.Printf("\rDownloading... %s Complete", humanize.Bytes(total))
	})

	fmt.Println("")
	if err != nil {
		t.Fatal(err)
	}
	assert.FileExists(t, tmpfile)
	os.Remove(tmpfile)
}

func TestGithubDownloadFail(t *testing.T) {
	repo, err := NewRepo("github.com/yaoapp/repo-does-not-exists", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Download("3bcaf04ae28f", nil)
	assert.Error(t, err, "404 Not Found")

	repo, err = NewRepo("github.com/yaoapp/demo-crm", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Download("ref-does-not-exists", nil)
	assert.Error(t, err, "404 Not Found")
}
