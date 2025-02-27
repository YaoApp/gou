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

func TestProcessList(t *testing.T) {
	prepare(t)
	defer clean()

	// Test with default options
	p := process.New("model.list", map[string]interface{}{})
	result, err := p.Exec()
	assert.Nil(t, err)
	models := result.([]map[string]interface{})
	assert.Greater(t, len(models), 0)

	// Check structure of returned model data
	for _, model := range models {
		assert.Contains(t, model, "id")
		assert.Contains(t, model, "name")
		assert.Contains(t, model, "description")
		assert.Contains(t, model, "file")
		assert.Contains(t, model, "table")
		assert.Contains(t, model, "primary")
		assert.NotContains(t, model, "metadata")
		assert.NotContains(t, model, "columns")
	}

	// Test with metadata option
	p = process.New("model.list", map[string]interface{}{"metadata": true})
	result, err = p.Exec()
	assert.Nil(t, err)
	models = result.([]map[string]interface{})
	assert.Greater(t, len(models), 0)

	// Check that metadata is included
	for _, model := range models {
		assert.Contains(t, model, "metadata")
		assert.NotContains(t, model, "columns")
	}

	// Test with columns option
	p = process.New("model.list", map[string]interface{}{"columns": true})
	result, err = p.Exec()
	assert.Nil(t, err)
	models = result.([]map[string]interface{})
	assert.Greater(t, len(models), 0)

	// Check that columns are included
	for _, model := range models {
		assert.Contains(t, model, "columns")
		assert.NotContains(t, model, "metadata")
	}

	// Test with both options
	p = process.New("model.list", map[string]interface{}{"metadata": true, "columns": true})
	result, err = p.Exec()
	assert.Nil(t, err)
	models = result.([]map[string]interface{})
	assert.Greater(t, len(models), 0)

	// Check that both metadata and columns are included
	for _, model := range models {
		assert.Contains(t, model, "metadata")
		assert.Contains(t, model, "columns")
	}
}
