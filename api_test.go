package gou

import "testing"

func TestLoadAPI(t *testing.T) {
	user := LoadAPI("user")
	user.Reload()
}
