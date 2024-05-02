package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestProcessLoadAndExists(t *testing.T) {
	prepare(t)
	defer clean()
	root := os.Getenv("GOU_TEST_APPLICATION")
	file := filepath.Join(root, "models", "user", "pet.mod.yao")

	source := `{
		"name": "TmpCategory",
		"table": { "name": "tmp_category", "comment": "Category" },
		"columns": [
		  { "label": "ID", "name": "id", "type": "ID" },
		  {
			"label": "Name",
			"name": "name",
			"type": "string",
			"length": 256,
			"comment": "Name",
			"index": true,
			"nullable": true
		  },
		  {
			"label": "Parent ID",
			"name": "parent_id",
			"type": "bigInteger",
			"index": true,
			"nullable": true
		  }
		],
		"relations": {},
		"values": [],
		"indexes": [],
		"option": { "timestamps": true, "soft_deletes": true }
	}`

	// Load User Pet
	p := process.New("models.user.pet.load", file)
	_, err := p.Exec()
	assert.Nil(t, err)

	p = process.New("models.user.pet.exists")
	result, err := p.Exec()
	assert.Nil(t, err)
	assert.True(t, result.(bool))

	p = process.New("models.tmpcategory.load", "<source>.mod.yao", source)
	_, err = p.Exec()
	assert.Nil(t, err)

	p = process.New("models.tmpcategory.exists")
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.True(t, result.(bool))

	p = process.New("models.not_found.exists")
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.False(t, result.(bool))

}

func TestProcessReload(t *testing.T) {
	prepare(t)
	defer clean()

	p := process.New("models.user.reload")
	_, err := p.Exec()
	assert.Nil(t, err)

	p = process.New("models.not-found.reload")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessRead(t *testing.T) {
	prepare(t)
	defer clean()

	p := process.New("models.user.read")
	data, err := p.Exec()
	assert.Nil(t, err)
	assert.Equal(t, data.(MetaData).Name, "User")

	p = process.New("models.not-found.read")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessMigrate(t *testing.T) {
	prepare(t)
	defer clean()

	p := process.New("models.user.migrate")
	_, err := p.Exec()
	assert.Nil(t, err)

	p = process.New("models.not-found.migrate")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessFind(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	p := process.New("models.user.find", 1, QueryParam{})
	_, err := p.Exec()
	assert.Nil(t, err)
}

func TestProcessGet(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	p := process.New("models.user.get", QueryParam{})
	_, err := p.Exec()
	assert.Nil(t, err)

}

func TestProcessPaginate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	p := process.New("models.user.paginate", QueryParam{}, 1, 2)
	_, err := p.Exec()
	assert.Nil(t, err)
}
