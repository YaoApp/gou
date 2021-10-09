package query

import (
	"testing"

	"github.com/yaoapp/xun/capsule"
)

func TestGet(t *testing.T) {
	dsl := MakeGou([]byte{}, capsule.Query())
	dsl.Get()
}

func TestFirst(t *testing.T) {
	dsl := MakeGou([]byte{}, capsule.Query())
	dsl.First()
}

func TestPaginate(t *testing.T) {
	dsl := MakeGou([]byte{}, capsule.Query())
	dsl.Paginate()
}

func TestRun(t *testing.T) {
	dsl := MakeGou([]byte{}, capsule.Query())
	dsl.First()
}
