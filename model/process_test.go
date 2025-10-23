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

func TestProcessMetadata(t *testing.T) {
	prepare(t)
	defer clean()

	p := process.New("models.user.metadata")
	data, err := p.Exec()
	assert.Nil(t, err)
	assert.Equal(t, data.(MetaData).Name, "User")

	p = process.New("models.not-found.metadata")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessRead(t *testing.T) {
	prepare(t)
	defer clean()

	p := process.New("models.user.read")
	data, err := p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, data)

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

func TestProcessCount(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	// Test count all records
	p := process.New("models.pet.count", QueryParam{})
	result, err := p.Exec()
	assert.Nil(t, err)
	count := result.(int)
	assert.Equal(t, 4, count)

	// Test count with conditions
	p = process.New("models.pet.count", QueryParam{
		Wheres: []QueryWhere{
			{Column: "name", Value: "Tommy", OP: "eq"},
		},
	})
	result, err = p.Exec()
	assert.Nil(t, err)
	count = result.(int)
	assert.Equal(t, 1, count)

	// Test count users
	p = process.New("models.user.count", QueryParam{})
	result, err = p.Exec()
	assert.Nil(t, err)
	count = result.(int)
	assert.Equal(t, 2, count)

	// Test count with multiple conditions
	p = process.New("models.pet.count", QueryParam{
		Wheres: []QueryWhere{
			{Column: "category_id", Value: 1, OP: "eq"},
		},
	})
	result, err = p.Exec()
	assert.Nil(t, err)
	count = result.(int)
	assert.GreaterOrEqual(t, count, 0)
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

func TestProcessDSL(t *testing.T) {
	prepare(t)
	defer clean()

	// Test with default options (no metadata or columns)
	p := process.New("model.dsl", "user", map[string]interface{}{})
	result, err := p.Exec()
	assert.Nil(t, err)
	modelData := result.(map[string]interface{})

	// Check basic structure
	assert.Equal(t, "user", modelData["id"])
	assert.NotContains(t, modelData, "metadata")
	assert.NotContains(t, modelData, "columns")

	// Test with metadata option
	p = process.New("model.dsl", "user", map[string]interface{}{"metadata": true})
	result, err = p.Exec()
	assert.Nil(t, err)
	modelData = result.(map[string]interface{})

	// Check that metadata is included
	assert.Equal(t, "user", modelData["id"])
	assert.Contains(t, modelData, "metadata")
	assert.NotContains(t, modelData, "columns")

	// Test with columns option
	p = process.New("model.dsl", "user", map[string]interface{}{"columns": true})
	result, err = p.Exec()
	assert.Nil(t, err)
	modelData = result.(map[string]interface{})

	// Check that columns are included
	assert.Equal(t, "user", modelData["id"])
	assert.Contains(t, modelData, "columns")
	assert.NotContains(t, modelData, "metadata")

	// Test with both options
	p = process.New("model.dsl", "user", map[string]interface{}{"metadata": true, "columns": true})
	result, err = p.Exec()
	assert.Nil(t, err)
	modelData = result.(map[string]interface{})

	// Check that both metadata and columns are included
	assert.Equal(t, "user", modelData["id"])
	assert.Contains(t, modelData, "metadata")
	assert.Contains(t, modelData, "columns")

	// Test with non-existent model
	p = process.New("model.dsl", "non_existent_model", map[string]interface{}{})
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessModelExists(t *testing.T) {
	prepare(t)
	defer clean()

	// Test with an existing model
	p := process.New("model.exists", "user")
	result, err := p.Exec()
	assert.Nil(t, err)
	exists := result.(bool)
	assert.True(t, exists)

	// Test with a non-existent model
	p = process.New("model.exists", "non_existent_model")
	result, err = p.Exec()
	assert.Nil(t, err)
	exists = result.(bool)
	assert.False(t, exists)
}

func TestProcessModelReload(t *testing.T) {
	prepare(t)
	defer clean()

	// Test with an existing model
	p := process.New("model.reload", "user")
	_, err := p.Exec()
	assert.Nil(t, err)

	// Test with a non-existent model
	p = process.New("model.reload", "non_existent_model")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessModelMigrate(t *testing.T) {
	prepare(t)
	defer clean()

	// Test with default option (false)
	p := process.New("model.migrate", "user")
	_, err := p.Exec()
	assert.Nil(t, err)

	// Test with explicit option (true)
	p = process.New("model.migrate", "user", true)
	_, err = p.Exec()
	assert.Nil(t, err)

	// Test with explicit option (false)
	p = process.New("model.migrate", "user", false)
	_, err = p.Exec()
	assert.Nil(t, err)

	// Test with a non-existent model
	p = process.New("model.migrate", "non_existent_model")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessModelLoad(t *testing.T) {
	prepare(t)
	defer clean()

	source := `{
		"name": "TestModel",
		"table": { "name": "test_model", "comment": "Test Model" },
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
		  }
		],
		"relations": {},
		"values": [],
		"indexes": [],
		"option": { "timestamps": true, "soft_deletes": true }
	}`

	// Test loading a new model
	p := process.New("model.load", "test_model", source)
	result, err := p.Exec()
	assert.Nil(t, err)
	if resultErr, ok := result.(error); ok {
		assert.Nil(t, resultErr)
	}

	// Verify the model was loaded
	p = process.New("model.exists", "test_model")
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.True(t, result.(bool))

	// Test loading with invalid source
	p = process.New("model.load", "invalid_model", "invalid json")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessModelUnload(t *testing.T) {
	prepare(t)
	defer clean()

	// First make sure the model exists
	p := process.New("model.exists", "user")
	result, err := p.Exec()
	assert.Nil(t, err)
	assert.True(t, result.(bool))

	// Unload the model
	p = process.New("model.unload", "user")
	_, err = p.Exec()
	assert.Nil(t, err)

	// Verify the model was unloaded
	p = process.New("model.exists", "user")
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.False(t, result.(bool))

	// Test unloading a non-existent model (should not error)
	p = process.New("model.unload", "non_existent_model")
	_, err = p.Exec()
	assert.Nil(t, err)
}

func TestProcessTakeSnapshot(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	p := process.New("models.user.takesnapshot", false)
	name, err := p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, name)
	assert.Contains(t, name.(string), "user_snapshot_")

	// Drop the snapshot table
	p = process.New("models.user.dropsnapshot", name)
	_, err = p.Exec()
	assert.Nil(t, err)
}

func TestProcessSnapshotExists(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	// Create a snapshot first
	p := process.New("models.user.takesnapshot", false)
	name, err := p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, name)

	// Test snapshot exists
	p = process.New("models.user.snapshotexists", name)
	exists, err := p.Exec()
	assert.Nil(t, err)
	assert.True(t, exists.(bool))

	// Drop the snapshot table
	p = process.New("models.user.dropsnapshot", name)
	_, err = p.Exec()
	assert.Nil(t, err)
}

func TestProcessRestoreSnapshot(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	// Create a snapshot first
	p := process.New("models.user.takesnapshot", false)
	name, err := p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, name)

	// Test restore snapshot
	p = process.New("models.user.restoresnapshot", name)
	_, err = p.Exec()
	assert.Nil(t, err)

	// Drop the snapshot table
	p = process.New("models.user.dropsnapshot", name)
	_, err = p.Exec()
	assert.Nil(t, err)
}

func TestProcessRestoreSnapshotByRename(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	// Create a snapshot first
	p := process.New("models.user.takesnapshot", false)
	name, err := p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, name)

	// Test restore snapshot by rename
	p = process.New("models.user.restoresnapshotbyrename", name)
	_, err = p.Exec()
	assert.Nil(t, err)
}

func TestProcessDropSnapshot(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	// Create a snapshot first
	p := process.New("models.user.takesnapshot", false)
	name, err := p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, name)

	// Test drop snapshot
	p = process.New("models.user.dropsnapshot", name)
	_, err = p.Exec()
	assert.Nil(t, err)

	// Verify snapshot was dropped
	p = process.New("models.user.snapshotexists", name)
	exists, err := p.Exec()
	assert.Nil(t, err)
	assert.False(t, exists.(bool))
}

func TestProcessSnapshotErrors(t *testing.T) {
	prepare(t)
	defer clean()

	// Test snapshot exists with invalid name
	p := process.New("models.user.snapshotexists", "invalid_snapshot")
	exists, err := p.Exec()
	assert.Nil(t, err)
	assert.False(t, exists.(bool))

	// Test restore snapshot with non-existent snapshot
	p = process.New("models.user.restoresnapshot", "invalid_snapshot")
	_, err = p.Exec()
	assert.NotNil(t, err)

	// Test restore snapshot by rename with non-existent snapshot
	p = process.New("models.user.restoresnapshotbyrename", "invalid_snapshot")
	_, err = p.Exec()
	assert.NotNil(t, err)

	// Test drop snapshot with invalid name
	p = process.New("models.user.dropsnapshot", "invalid_snapshot")
	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestProcessUpsert(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	// Test upsert with string uniqueBy parameter - create a new record
	p := process.New("models.user.upsert", map[string]interface{}{
		"name":   "Upsert User 1",
		"mobile": "13900001111",
		"status": "enabled",
	}, "mobile")
	result, err := p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// Update the record with the same mobile
	p = process.New("models.user.upsert", map[string]interface{}{
		"name":   "Upsert User 1 Updated",
		"mobile": "13900001111",
		"status": "enabled",
	}, "mobile")
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// Test upsert with string array uniqueBy parameter
	p = process.New("models.user.upsert", map[string]interface{}{
		"name":   "Upsert User 2",
		"mobile": "13900002222",
		"status": "enabled",
	}, []string{"mobile"})
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// Test upsert with interface array uniqueBy parameter
	p = process.New("models.user.upsert", map[string]interface{}{
		"name":   "Upsert User 3",
		"mobile": "13900003333",
		"status": "enabled",
	}, []interface{}{"mobile"})
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// Test upsert with updateColumns parameter - create a new record
	p = process.New("models.user.upsert", map[string]interface{}{
		"name":   "Upsert User 4",
		"mobile": "13900004444",
		"status": "enabled",
	}, "mobile", []string{"name", "status"})
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// Update with specific columns
	p = process.New("models.user.upsert", map[string]interface{}{
		"name":   "Upsert User 4 Updated",
		"mobile": "13900004444",
		"status": "disabled",
	}, "mobile", []string{"name", "status"})
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// Test upsert with interface array updateColumns parameter
	p = process.New("models.user.upsert", map[string]interface{}{
		"name":   "Upsert User 5",
		"mobile": "13900005555",
		"status": "enabled",
	}, "mobile", []interface{}{"name", "status"})
	result, err = p.Exec()
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// Test upsert with invalid uniqueBy parameter
	p = process.New("models.user.upsert", map[string]interface{}{
		"name": "User with Invalid UniqueBy",
	}, []int{1, 2})
	_, err = p.Exec()
	assert.NotNil(t, err)

	// Test upsert with empty uniqueBy parameter
	p = process.New("models.user.upsert", map[string]interface{}{
		"name": "User with Empty UniqueBy",
	}, []string{})
	_, err = p.Exec()
	assert.NotNil(t, err)

	// Test upsert with invalid updateColumns parameter
	p = process.New("models.user.upsert", map[string]interface{}{
		"name":   "User with Invalid UpdateColumns",
		"mobile": "13900006666",
		"status": "enabled",
	}, "mobile", 123)
	_, err = p.Exec()
	assert.NotNil(t, err)
}
