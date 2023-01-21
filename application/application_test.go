package application

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenFromDisk(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	_, err := OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = OpenFromDisk("/path/not-exists")
	assert.NotNil(t, err)
}
