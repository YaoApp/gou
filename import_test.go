package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImportInsertCSV(t *testing.T) {

	manucsv := NewImport(20).Using(map[string]string{
		"name":       "name",
		"short_name": "smalltitle",
		"rank":       "priority",
	}).Set("strict", true)

	manu := Select("manu")
	err := manu.Migrate(true)
	if err != nil {
		t.Fatal(err)
	}

	csvfile := path.Join(TestModRoot, "manu.csv")
	res := manucsv.InsertCSV(csvfile, "manu")
	assert.Equal(t, 243, res.Total)
	assert.Equal(t, 143, res.Success)
	assert.Equal(t, 100, res.Failure)
	err = manu.Migrate(true)
	if err != nil {
		t.Fatal(err)
	}
}
