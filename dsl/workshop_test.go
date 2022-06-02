package dsl

import (
	"fmt"
	"os"
	"testing"
)

func TestOpenWorkshop(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	_, err := OpenWorkshop(root)
	fmt.Println(err)
}
