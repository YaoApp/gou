package query

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
)

func TestGouFile(t *testing.T) {
	assert.Panics(t, func() {
		GouFile(path.Join(TestQueryRoot, "not-exists.json"))
	})
	gou := GouFile(path.Join(TestQueryRoot, "full.json"))
	utils.Dump(gou.QueryDSL)
}

func TestGet(t *testing.T) {
	rows := Gou([]byte{}).With(qb).Get()
	utils.Dump(rows)
}

func TestFirst(t *testing.T) {
	Gou([]byte{}).With(qb).First()
}

func TestPaginate(t *testing.T) {
	Gou([]byte{}).With(qb).Paginate()
}

func TestRun(t *testing.T) {
	Gou([]byte{}).With(qb).Run()
}
