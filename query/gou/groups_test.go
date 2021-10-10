package gou

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestGroupToMap(t *testing.T) {

	group := Group{Field: NewExpression("type")}
	assert.Equal(t, map[string]interface{}{"field": "type"}, group.ToMap())

	group = Group{Field: NewExpression("type"), Rollup: "", Comment: ""}
	assert.Equal(t, map[string]interface{}{"field": "type"}, group.ToMap())

	group = Group{Field: NewExpression("id"), Rollup: "城市"}
	assert.Equal(t, map[string]interface{}{"field": "id", "rollup": "城市"}, group.ToMap())

	group = Group{Field: NewExpression("type"), Comment: "按类型"}
	assert.Equal(t, map[string]interface{}{"field": "type", "comment": "按类型"}, group.ToMap())
}

func TestGroupsStrict(t *testing.T) {
	var stricts []Groups
	bytes := ReadFile("groups/strict.json")

	err := jsoniter.Unmarshal(bytes, &stricts)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(stricts))

	// [{ "field": "kind" }]
	assert.Equal(t, "kind", stricts[0][0].Field.ToString())
	assert.Equal(t, "", stricts[0][0].Rollup)

	// [{ "field": "city", "rollup": "所有城市" }],
	assert.Equal(t, "city", stricts[1][0].Field.ToString())
	assert.Equal(t, "所有城市", stricts[1][0].Rollup)

	// [{ "field": "@industries" }],
	assert.Equal(t, "@industries", stricts[2][0].Field.ToString())
	assert.Equal(t, "", stricts[2][0].Rollup)

	// [{ "field": "kind" }, { "field": "city" }],
	assert.Equal(t, "kind", stricts[3][0].Field.ToString())
	assert.Equal(t, "", stricts[3][0].Rollup)
	assert.Equal(t, "city", stricts[3][1].Field.ToString())
	assert.Equal(t, "", stricts[3][1].Rollup)

	// [
	//   { "field": "kind", "rollup": "所有类型", "comment": "按类型统计" },
	//   { "field": "city", "comment": "按城市统计" }
	// ],
	assert.Equal(t, "kind", stricts[4][0].Field.ToString())
	assert.Equal(t, "所有类型", stricts[4][0].Rollup)
	assert.Equal(t, "按类型统计", stricts[4][0].Comment)
	assert.Equal(t, "city", stricts[4][1].Field.ToString())
	assert.Equal(t, "", stricts[4][1].Rollup)
	assert.Equal(t, "按城市统计", stricts[4][1].Comment)

	// [
	//   { "field": "kind", "rollup": "所有类型", "comment": "按类型统计" },
	//   { "field": "city", "rollup": "所有城市", "comment": "按城市统计" }
	// ]
	assert.Equal(t, "kind", stricts[5][0].Field.ToString())
	assert.Equal(t, "所有类型", stricts[5][0].Rollup)
	assert.Equal(t, "按类型统计", stricts[5][0].Comment)
	assert.Equal(t, "city", stricts[5][1].Field.ToString())
	assert.Equal(t, "所有城市", stricts[5][1].Rollup)
	assert.Equal(t, "按城市统计", stricts[5][1].Comment)
}

func TestGroupsSugar(t *testing.T) {
	var sugars []Groups
	bytes := ReadFile("groups/sugar.json")

	err := jsoniter.Unmarshal(bytes, &sugars)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(sugars))

	// "kind",
	assert.Equal(t, "kind", sugars[0][0].Field.ToString())
	assert.Equal(t, "", sugars[0][0].Rollup)

	// "city rollup 所有城市",
	assert.Equal(t, "city", sugars[1][0].Field.ToString())
	assert.Equal(t, "所有城市", sugars[1][0].Rollup)

	// "@industries",
	assert.Equal(t, "@industries", sugars[2][0].Field.ToString())
	assert.Equal(t, "", sugars[2][0].Rollup)

	// "kind,city"
	assert.Equal(t, "kind", sugars[3][0].Field.ToString())
	assert.Equal(t, "", sugars[3][0].Rollup)
	assert.Equal(t, "city", sugars[3][1].Field.ToString())
	assert.Equal(t, "", sugars[3][1].Rollup)

	// "kind rollup 所有类型, city",
	assert.Equal(t, "kind", sugars[4][0].Field.ToString())
	assert.Equal(t, "所有类型", sugars[4][0].Rollup)
	assert.Equal(t, "city", sugars[4][1].Field.ToString())
	assert.Equal(t, "", sugars[4][1].Rollup)

	// "kind rollup 所有类型, city rollup 所有城市"
	assert.Equal(t, "kind", sugars[5][0].Field.ToString())
	assert.Equal(t, "所有类型", sugars[5][0].Rollup)
	assert.Equal(t, "city", sugars[5][1].Field.ToString())
	assert.Equal(t, "所有城市", sugars[5][1].Rollup)
}

func TestGroupsMix(t *testing.T) {
	var mixes []Groups
	bytes := ReadFile("groups/mix.json")

	err := jsoniter.Unmarshal(bytes, &mixes)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(mixes))

	// "kind",
	assert.Equal(t, "kind", mixes[0][0].Field.ToString())
	assert.Equal(t, "", mixes[0][0].Rollup)

	// "city rollup 所有城市",
	assert.Equal(t, "city", mixes[1][0].Field.ToString())
	assert.Equal(t, "所有城市", mixes[1][0].Rollup)

	// "@industries",
	assert.Equal(t, "@industries", mixes[2][0].Field.ToString())
	assert.Equal(t, "", mixes[2][0].Rollup)

	// "kind,city"
	assert.Equal(t, "kind", mixes[3][0].Field.ToString())
	assert.Equal(t, "", mixes[3][0].Rollup)
	assert.Equal(t, "city", mixes[3][1].Field.ToString())
	assert.Equal(t, "", mixes[3][1].Rollup)

	// "kind rollup 所有类型, city",
	assert.Equal(t, "kind", mixes[4][0].Field.ToString())
	assert.Equal(t, "所有类型", mixes[4][0].Rollup)
	assert.Equal(t, "city", mixes[4][1].Field.ToString())
	assert.Equal(t, "", mixes[4][1].Rollup)

	// "kind rollup 所有类型, city rollup 所有城市"
	assert.Equal(t, "kind", mixes[5][0].Field.ToString())
	assert.Equal(t, "所有类型", mixes[5][0].Rollup)
	assert.Equal(t, "city", mixes[5][1].Field.ToString())
	assert.Equal(t, "所有城市", mixes[5][1].Rollup)

	// ["city"]
	assert.Equal(t, "city", mixes[6][0].Field.ToString())
	assert.Equal(t, "", mixes[6][0].Rollup)

	// ["kind", "city"],
	assert.Equal(t, "kind", mixes[7][0].Field.ToString())
	assert.Equal(t, "", mixes[7][0].Rollup)
	assert.Equal(t, "city", mixes[7][1].Field.ToString())
	assert.Equal(t, "", mixes[7][1].Rollup)

	// ["kind", { "field": "city", "rollup": "所有类型", "comment": "按类型统计" }],
	assert.Equal(t, "kind", mixes[8][0].Field.ToString())
	assert.Equal(t, "", mixes[8][0].Rollup)
	assert.Equal(t, "city", mixes[8][1].Field.ToString())
	assert.Equal(t, "所有类型", mixes[8][1].Rollup)
	assert.Equal(t, "按类型统计", mixes[8][1].Comment)

	// [{ "field": "@industries", "rollup": "所有行业", "comment": "按行业统计" }]
	assert.Equal(t, "@industries", mixes[9][0].Field.ToString())
	assert.Equal(t, "所有行业", mixes[9][0].Rollup)
	assert.Equal(t, "按行业统计", mixes[9][0].Comment)

}

func TestGroupsValidate(t *testing.T) {
	var errs []Groups
	bytes := ReadFile("groups/error.json")

	err := jsoniter.Unmarshal(bytes, &errs)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(errs))

	// [{ "fields": "id" }],
	assert.Nil(t, errs[0])

	// "id rolslup",
	res := errs[1].Validate()
	assert.Equal(t, 1, len(res))
	assert.Contains(t, res[0].Error(), "(id rolslup)")

	// [{ "field": "id" }, "type rolslup"]
	res = errs[2].Validate()
	assert.Equal(t, 1, len(res))
	assert.Contains(t, res[0].Error(), "(type rolslup)")
	assert.Contains(t, res[0].Error(), "2")

	groups := Groups{
		{Rollup: "按时间排序"},
	}
	res = groups.Validate()
	assert.Equal(t, 1, len(res))
	assert.Contains(t, res[0].Error(), "field")
	assert.Contains(t, res[0].Error(), "1")

}

func TestGroupsUnmarshalJSONError(t *testing.T) {
	var groups Groups
	err := jsoniter.Unmarshal([]byte(`{1}`), &groups)
	assert.NotNil(t, err)
}

func TestGroupsMarshalJSON(t *testing.T) {
	group := Group{Field: NewExpression("id")}
	bytes, err := jsoniter.Marshal(group)
	assert.Nil(t, err)
	assert.Contains(t, string(bytes), `"field":"id"`)
	assert.NotContains(t, string(bytes), `"rollup"`)

	group = Group{Field: NewExpression("id"), Rollup: "全部ID", Comment: "Unit-Test"}
	bytes, err = jsoniter.Marshal(group)
	assert.Nil(t, err)
	assert.Contains(t, string(bytes), `"field":"id"`)
	assert.Contains(t, string(bytes), `"rollup":"全部ID"`)
	assert.Contains(t, string(bytes), `"comment":"Unit-Test"`)

	groups := Groups{
		{Field: NewExpression("id")},
		{Field: NewExpression("type"), Rollup: "全部类型"},
		{Field: NewExpression("@industries"), Rollup: "全部行业", Comment: "按行业"},
	}
	bytes, err = jsoniter.Marshal(groups)
	assert.Nil(t, err)
	assert.Contains(t, string(bytes), `"field":"id"`)
	assert.NotContains(t, string(bytes), `"rollup":""`)
	assert.Contains(t, string(bytes), `"field":"type"`)
	assert.Contains(t, string(bytes), `"field":"@industries"`)
	assert.Contains(t, string(bytes), `"rollup":"全部类型"`)
	assert.Contains(t, string(bytes), `"rollup":"全部行业"`)
	assert.Contains(t, string(bytes), `"comment":"按行业"`)
}

func TestGroupsPushStringError(t *testing.T) {
	groups := Groups{}
	err := groups.PushString("a rollup b rollup c")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), `a rollup b rollup c`)
}
