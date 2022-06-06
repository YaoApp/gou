package workshop

import (
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestGetConfig(t *testing.T) {
	cfg, err := GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	token := os.Getenv("GOU_TEST_GITHUB_TOKEN")
	assert.Equal(t, token, cfg["github.com"]["token"])
}
