package gou

import "testing"

func TestLoadModel(t *testing.T) {
	user := LoadModel("user")
	user.Migrate()
}
