package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

func TestTakeSnapshotInMemory(t *testing.T) {
	prepare(t)
	defer clean()
	pet := Select("pet")
	id, err := pet.Save(maps.Map{"name": "Cookie"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, any.Of(id).CInt())

	name, err := pet.TakeSnapshot(true)
	assert.Nil(t, err)
	assert.Contains(t, name, "pet_snapshot_")

	// Drop the table
	err = pet.DropSnapshotTable(name)
	assert.Nil(t, err)
}

func TestTakeSnapshot(t *testing.T) {
	prepare(t)
	defer clean()
	pet := Select("pet")
	id, err := pet.Save(maps.Map{"name": "Cookie"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, any.Of(id).CInt())

	name, err := pet.TakeSnapshot(false)
	assert.Nil(t, err)
	assert.Contains(t, name, "pet_snapshot_")

	// Drop the table
	err = pet.DropSnapshotTable(name)
	assert.Nil(t, err)
}

func TestRestoreSnapshot(t *testing.T) {
	prepare(t)
	defer clean()
	pet := Select("pet")
	id, err := pet.Save(maps.Map{"name": "Cookie"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, any.Of(id).CInt())

	name, err := pet.TakeSnapshot(false)
	assert.Nil(t, err)
	assert.Contains(t, name, "pet_snapshot_")

	// Restore the snapshot
	err = pet.RestoreSnapshot(name)
	assert.Nil(t, err)

	// Check the snapshot
	pet = Select("pet")
	row, err := pet.Find(1, QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, "Cookie", row.Get("name"))

	// Drop the snapshot table
	err = pet.DropSnapshotTable(name)
	assert.Nil(t, err)
}

func TestRestoreSnapshotWithInMemory(t *testing.T) {
	prepare(t)
	defer clean()
	pet := Select("pet")
	id, err := pet.Save(maps.Map{"name": "Cookie"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, any.Of(id).CInt())

	name, err := pet.TakeSnapshot(true)
	assert.Nil(t, err)
	assert.Contains(t, name, "pet_snapshot_")

	// Restore the snapshot
	err = pet.RestoreSnapshot(name)
	assert.Nil(t, err)

	// Check the snapshot
	pet = Select("pet")
	row, err := pet.Find(1, QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, "Cookie", row.Get("name"))

	// Drop the snapshot table
	err = pet.DropSnapshotTable(name)
	assert.Nil(t, err)

}

func TestRestoreSnapshotByRename(t *testing.T) {
	prepare(t)
	defer clean()
	pet := Select("pet")
	id, err := pet.Save(maps.Map{"name": "Cookie"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, any.Of(id).CInt())

	name, err := pet.TakeSnapshot(true)
	assert.Nil(t, err)
	assert.Contains(t, name, "pet_snapshot_")

	// Restore the snapshot
	err = pet.RestoreSnapshotByRename(name)
	assert.Nil(t, err)

	// Check the snapshot
	pet = Select("pet")
	row, err := pet.Find(1, QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, "Cookie", row.Get("name"))

	// Drop the snapshot table
	err = pet.DropSnapshotTable(name)
	assert.Nil(t, err)
}
