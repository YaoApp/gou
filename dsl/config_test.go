package dsl

import (
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestConfig(t *testing.T) {
	cfg, err := Config()
	if err != nil {
		t.Fatal(err)
	}
	token := os.Getenv("GOU_TEST_GITHUB_TOKEN")
	assert.Equal(t, token, cfg["github.com"]["token"])
}
