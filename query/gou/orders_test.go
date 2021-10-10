package gou

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestOrdersStrict(t *testing.T) {
	var stricts []Orders
	bytes := ReadFile("orders/strict.json")

	err := jsoniter.Unmarshal(bytes, &stricts)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(stricts))

	// [{ "field": "id" }]
	assert.Equal(t, "id", stricts[0][0].Field.ToString())
	assert.Equal(t, "asc", stricts[0][0].Sort)

	// [{ "field": "id", "sort": "desc" }],
	assert.Equal(t, "id", stricts[1][0].Field.ToString())
	assert.Equal(t, "desc", stricts[1][0].Sort)

	// [{ "field": "type" }, { "field": "created_at", "sort": "asc" }],
	assert.Equal(t, "type", stricts[2][0].Field.ToString())
	assert.Equal(t, "asc", stricts[2][0].Sort)
	assert.Equal(t, "created_at", stricts[2][1].Field.ToString())
	assert.Equal(t, "asc", stricts[2][1].Sort)

	// [{ "field": "type", "sort": "desc" }, { "field": "created_at", "sort": "asc" }],
	assert.Equal(t, "type", stricts[3][0].Field.ToString())
	assert.Equal(t, "desc", stricts[3][0].Sort)
	assert.Equal(t, "created_at", stricts[3][1].Field.ToString())
	assert.Equal(t, "asc", stricts[3][1].Sort)

	// [{ "field": "type", "sort": "desc" }, { "field": "created_at" }]
	assert.Equal(t, "type", stricts[4][0].Field.ToString())
	assert.Equal(t, "desc", stricts[4][0].Sort)
	assert.Equal(t, "created_at", stricts[4][1].Field.ToString())
	assert.Equal(t, "asc", stricts[4][1].Sort)
}

func TestOrdersSugar(t *testing.T) {
	var sugars []Orders
	bytes := ReadFile("orders/sugar.json")

	err := jsoniter.Unmarshal(bytes, &sugars)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(sugars))

	// "id",
	assert.Equal(t, "id", sugars[0][0].Field.ToString())
	assert.Equal(t, "asc", sugars[0][0].Sort)

	//  "id desc",
	assert.Equal(t, "id", sugars[1][0].Field.ToString())
	assert.Equal(t, "desc", sugars[1][0].Sort)

	//  "type asc",
	assert.Equal(t, "type", sugars[2][0].Field.ToString())
	assert.Equal(t, "asc", sugars[2][0].Sort)

	// "id , type desc",
	assert.Equal(t, "id", sugars[3][0].Field.ToString())
	assert.Equal(t, "asc", sugars[3][0].Sort)
	assert.Equal(t, "type", sugars[3][1].Field.ToString())
	assert.Equal(t, "desc", sugars[3][1].Sort)

	// "id desc, type",
	assert.Equal(t, "id", sugars[4][0].Field.ToString())
	assert.Equal(t, "desc", sugars[4][0].Sort)
	assert.Equal(t, "type", sugars[4][1].Field.ToString())
	assert.Equal(t, "asc", sugars[4][1].Sort)

	// "id desc , type desc"
	assert.Equal(t, "id", sugars[5][0].Field.ToString())
	assert.Equal(t, "desc", sugars[5][0].Sort)
	assert.Equal(t, "type", sugars[5][1].Field.ToString())
	assert.Equal(t, "desc", sugars[5][1].Sort)
}

func TestOrdersMix(t *testing.T) {
	var mixes []Orders
	bytes := ReadFile("orders/mix.json")

	err := jsoniter.Unmarshal(bytes, &mixes)
	assert.Nil(t, err)
	assert.Equal(t, 8, len(mixes))

	// "id",
	assert.Equal(t, "id", mixes[0][0].Field.ToString())
	assert.Equal(t, "asc", mixes[0][0].Sort)

	//  "id desc",
	assert.Equal(t, "id", mixes[1][0].Field.ToString())
	assert.Equal(t, "desc", mixes[1][0].Sort)

	//  "type asc",
	assert.Equal(t, "type", mixes[2][0].Field.ToString())
	assert.Equal(t, "asc", mixes[2][0].Sort)

	// "id , type desc",
	assert.Equal(t, "id", mixes[3][0].Field.ToString())
	assert.Equal(t, "asc", mixes[3][0].Sort)
	assert.Equal(t, "type", mixes[3][1].Field.ToString())
	assert.Equal(t, "desc", mixes[3][1].Sort)

	// "id desc, type",
	assert.Equal(t, "id", mixes[4][0].Field.ToString())
	assert.Equal(t, "desc", mixes[4][0].Sort)
	assert.Equal(t, "type", mixes[4][1].Field.ToString())
	assert.Equal(t, "asc", mixes[4][1].Sort)

	// "id desc , type desc"
	assert.Equal(t, "id", mixes[5][0].Field.ToString())
	assert.Equal(t, "desc", mixes[5][0].Sort)
	assert.Equal(t, "type", mixes[5][1].Field.ToString())
	assert.Equal(t, "desc", mixes[5][1].Sort)

	// [{ "field": "type", "sort": "desc" }, { "field": "created_at", "sort": "asc" }],
	assert.Equal(t, "type", mixes[6][0].Field.ToString())
	assert.Equal(t, "desc", mixes[6][0].Sort)
	assert.Equal(t, "created_at", mixes[6][1].Field.ToString())
	assert.Equal(t, "asc", mixes[6][1].Sort)

	// ["type desc", { "field": "created_at" }]
	assert.Equal(t, "type", mixes[7][0].Field.ToString())
	assert.Equal(t, "desc", mixes[7][0].Sort)
	assert.Equal(t, "created_at", mixes[7][1].Field.ToString())
	assert.Equal(t, "asc", mixes[7][1].Sort)

}

func TestOrdersValidate(t *testing.T) {
	var errs []Orders
	bytes := ReadFile("orders/error.json")

	err := jsoniter.Unmarshal(bytes, &errs)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(errs))

	// [{ "sort": "asc" }],
	// [{ "fields": "type", "sorts": "desc" }, { "sort": "asc" }],
	// "id error",
	// "id desc, name error  "

	// [{ "sort": "asc" }]
	assert.Nil(t, errs[0])

	// [{ "fields": "type", "sorts": "desc" }, { "sort": "asc" }],
	assert.Nil(t, errs[1])

	// "id error"
	res := errs[2].Validate()
	assert.Equal(t, 1, len(res))
	assert.Contains(t, res[0].Error(), "(error)")

	// "id desc, name error  "
	res = errs[3].Validate()
	assert.Equal(t, 1, len(res))
	assert.Contains(t, res[0].Error(), "(error)")
	assert.Contains(t, res[0].Error(), "2")

}
