package query

import (
	"testing"
)

func TestGet(t *testing.T) {
	dsl := MakeGou([]byte{})
	dsl.Get()
}

func TestFirst(t *testing.T) {
	dsl := MakeGou([]byte{})
	dsl.First()
}

func TestPaginate(t *testing.T) {
	dsl := MakeGou([]byte{})
	dsl.Paginate()
}

func TestRun(t *testing.T) {
	dsl := MakeGou([]byte{})
	dsl.First()
}
