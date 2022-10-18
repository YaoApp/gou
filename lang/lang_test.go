package lang

import (
	"os"
	"path/filepath"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

type TestSomeWidget struct {
	Flow  string
	Model string
	Hello string
	Foo   string
	Bar   string
}

type testVal struct {
	Name string
	Desc string
}

func (w *TestSomeWidget) Lang(trans func(widget string, inst string, value *string) bool) {
	trans("flow", "hello", &w.Flow)
	trans("model", "demo", &w.Model)
	trans("other", "inst-1", &w.Hello)
	trans("other", "inst-1", &w.Foo)
	trans("other", "inst-1", &w.Bar)
}

func TestLoadPick(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs")
	err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, Dicts, 2)

	dict := Pick("zh-cn")
	assert.Len(t, dict.Global, 5)
	assert.Len(t, dict.Widgets, 4)
	assert.Len(t, dict.Widgets["flow.hello"], 1)
	assert.Len(t, dict.Widgets["model.demo"], 1)
	assert.Equal(t, dict.Name, "zh-cn")
}

func TestLoadMerge(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs-tests")
	err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, Dicts, 2)

	dict := Pick("zh-cn")
	assert.Len(t, dict.Global, 6)
	assert.Len(t, dict.Widgets, 4)
	assert.Len(t, dict.Widgets["flow.hello"], 2)
	assert.Len(t, dict.Widgets["model.demo"], 2)
	assert.Equal(t, dict.Name, "zh-cn")
}

func TestApply(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs")
	err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	w := &TestSomeWidget{
		Model: "::SN",
		Flow:  "::Demo",
		Foo:   "::NotDefined",
		Bar:   "Ignore",
		Hello: "\\:\\:World",
	}

	w1 := *w
	dict := Pick("zh-cn")
	dict.Apply(&w1)
	assert.Equal(t, w1.Flow, "演示")
	assert.Equal(t, w1.Model, "编码")
	assert.Equal(t, w1.Hello, "::World")
	assert.Equal(t, w1.Foo, "NotDefined")
	assert.Equal(t, w1.Bar, "Ignore")

	w2 := *w
	dict = Pick("zh-hk")
	dict.Apply(&w2)
	assert.Equal(t, w2.Flow, "演示")
	assert.Equal(t, w2.Model, "編碼")
	assert.Equal(t, w2.Hello, "::World")
	assert.Equal(t, w2.Foo, "NotDefined")
	assert.Equal(t, w2.Bar, "Ignore")
}

func TestApplyMuti(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs")
	err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	root = os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs-tests")
	err = Load(root)
	if err != nil {
		t.Fatal(err)
	}

	w := &TestSomeWidget{
		Model: "$L(SN)",
		Flow:  "$L(Create)$L(Demo)",
		Foo:   "$L(NotDefined)",
		Bar:   "Ignore",
		Hello: "\\$L(Create) ::World",
	}

	w1 := *w
	dict := Pick("zh-cn")
	dict.Apply(&w1)
	assert.Equal(t, w1.Flow, "创建演示")
	assert.Equal(t, w1.Model, "编码")
	assert.Equal(t, w1.Hello, "$L(Create) ::World")
	assert.Equal(t, w1.Foo, "NotDefined")
	assert.Equal(t, w1.Bar, "Ignore")

	w2 := *w
	dict = Pick("zh-hk")
	dict.Apply(&w2)
	assert.Equal(t, w2.Flow, "創建演示")
	assert.Equal(t, w2.Model, "編碼")
	assert.Equal(t, w2.Hello, "$L(Create) ::World")
	assert.Equal(t, w2.Foo, "NotDefined")
	assert.Equal(t, w2.Bar, "Ignore")
}

func TestReplace(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs")
	err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	s := "ZipCode"
	Pick("zh-cn").AsDefault()

	s1 := s
	Replace(&s1)
	assert.Equal(t, s1, "邮政编码")

	Pick("zh-hk").AsDefault()
	s2 := s
	Replace(&s2)
	assert.Equal(t, s2, "郵政編碼")
}

func TestReplaceAll(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs")
	err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	data := testData()
	dict := Pick("zh-cn")

	var intv = data["int"]
	err = dict.ReplaceAll([]string{"model.demo"}, &intv)
	if err != nil {
		t.Fatal(err)
	}

	var floatv = data["float"]
	err = dict.ReplaceAll([]string{"model.demo"}, &floatv)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0.618, floatv)

	var s1 = data["s1"]
	err = dict.ReplaceAll([]string{"model.demo"}, &s1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "邮政编码", s1)

	var s2 = data["s2"]
	err = dict.ReplaceAll([]string{"model.demo"}, &s2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "编码", s2)

	var s3 = data["s3"]
	err = dict.ReplaceAll([]string{"model.demo"}, &s3)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Latest", s3)

	var s4 = data["s4"]
	err = dict.ReplaceAll([]string{"model.demo"}, &s4)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "S4: 邮政编码", s4)

	var s5 = data["s5"]
	err = dict.ReplaceAll([]string{"model.demo"}, &s5)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "S5: 编码", s5)

	var s6 = data["s6"]
	err = dict.ReplaceAll([]string{"model.demo"}, &s6)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "S6: Latest", s6)

	var s7 = data["s7"]
	err = dict.ReplaceAll([]string{"model.demo"}, &s7)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "S7: 邮政编码 编码 Latest", s7)

	var stru = data["struct"].(testVal)
	err = dict.ReplaceAll([]string{"model.demo"}, &stru)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "编码", stru.Name)
	assert.Equal(t, "ST: 邮政编码 编码 Latest", stru.Desc)

	var struptr = data["structptr"].(*testVal)
	err = dict.ReplaceAll([]string{"model.demo"}, &struptr)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "编码", struptr.Name)
	assert.Equal(t, "ST: 邮政编码 编码 Latest", struptr.Desc)

	var arr = data["arr"]
	err = dict.ReplaceAll([]string{"model.demo"}, &arr)
	if err != nil {
		t.Fatal(err)
	}
	res, err := jsoniter.Marshal(arr)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, string(res), "::")
	assert.NotContains(t, string(res), "$L")

	var mapv = data["map"]
	err = dict.ReplaceAll([]string{"model.demo"}, &mapv)
	if err != nil {
		t.Fatal(err)
	}
	res, err = jsoniter.Marshal(mapv)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, string(res), "::")
	assert.NotContains(t, string(res), "$L")

	err = dict.ReplaceAll([]string{"model.demo"}, &data)
	if err != nil {
		t.Fatal(err)
	}
	res, err = jsoniter.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, string(res), "::")
	assert.NotContains(t, string(res), "$L")
}

func TestReplaceClone(t *testing.T) {

	root := os.Getenv("GOU_TEST_APP_ROOT")
	root = filepath.Join(root, "langs")
	err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	data := testData()
	dict := Pick("zh-cn")

	var stru = data["struct"].(testVal)
	new, err := dict.ReplaceClone([]string{"model.demo"}, stru)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "::SN", stru.Name)
	assert.Equal(t, "ST: $L(ZipCode) $L(SN) $L(Latest)", stru.Desc)
	assert.Equal(t, "编码", new.(testVal).Name)
	assert.Equal(t, "ST: 邮政编码 编码 Latest", new.(testVal).Desc)

	new, err = dict.ReplaceClone([]string{"model.demo"}, &stru)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "::SN", stru.Name)
	assert.Equal(t, "ST: $L(ZipCode) $L(SN) $L(Latest)", stru.Desc)
	assert.Equal(t, "编码", new.(*testVal).Name)
	assert.Equal(t, "ST: 邮政编码 编码 Latest", new.(*testVal).Desc)

	var struptr = data["structptr"].(*testVal)
	new, err = dict.ReplaceClone([]string{"model.demo"}, struptr)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "::SN", struptr.Name)
	assert.Equal(t, "ST: $L(ZipCode) $L(SN) $L(Latest)", struptr.Desc)
	assert.Equal(t, "编码", new.(*testVal).Name)
	assert.Equal(t, "ST: 邮政编码 编码 Latest", new.(*testVal).Desc)

	var arr = data["arr"]
	new, err = dict.ReplaceClone([]string{"model.demo"}, arr)
	if err != nil {
		t.Fatal(err)
	}
	res, err := jsoniter.Marshal(arr)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(res), "::")
	assert.Contains(t, string(res), "$L")

	res, err = jsoniter.Marshal(new)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, string(res), "::")
	assert.NotContains(t, string(res), "$L")

	var mapv = data["map"]
	new, err = dict.ReplaceClone([]string{"model.demo"}, mapv)
	if err != nil {
		t.Fatal(err)
	}
	res, err = jsoniter.Marshal(mapv)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(res), "::")
	assert.Contains(t, string(res), "$L")

	res, err = jsoniter.Marshal(new)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, string(res), "::")
	assert.NotContains(t, string(res), "$L")

	new, err = dict.ReplaceClone([]string{"model.demo"}, data)
	if err != nil {
		t.Fatal(err)
	}

	res, err = jsoniter.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(res), "::")
	assert.Contains(t, string(res), "$L")

	res, err = jsoniter.Marshal(new)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, string(res), "::")
	assert.NotContains(t, string(res), "$L")
}

func testData() map[string]interface{} {
	return map[string]interface{}{
		"int":   1,
		"float": 0.618,
		"s1":    "::ZipCode",
		"s2":    "::SN",
		"s3":    "::Latest",
		"s4":    "S4: $L(ZipCode)",
		"s5":    "S5: $L(SN)",
		"s6":    "S6: $L(Latest)",
		"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
		"arr": []interface{}{
			1, 0.618,
			"::ZipCode", "::SN", "::Latest", "S4: $L(ZipCode)", "S5: $L(SN)", "S6: $L(Latest)",
			"S7: $L(ZipCode) $L(SN) $L(Latest)",

			[]interface{}{
				1, 0.618,
				"::ZipCode", "::SN", "::Latest", "S4: $L(ZipCode)", "S5: $L(SN)", "S6: $L(Latest)",
				"S7: $L(ZipCode) $L(SN) $L(Latest)",
				map[string]interface{}{
					"int":   1,
					"float": 0.618,
					"s1":    "::ZipCode",
					"s2":    "::SN",
					"s3":    "::Latest",
					"s4":    "S4: $L(ZipCode)",
					"s5":    "S5: $L(SN)",
					"s6":    "S6: $L(Latest)",
					"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
				},
				[]interface{}{
					1, 0.618,
					"::ZipCode", "::SN", "::Latest", "S4: $L(ZipCode)", "S5: $L(SN)", "S6: $L(Latest)",
					"S7: $L(ZipCode) $L(SN) $L(Latest)",
					map[string]interface{}{
						"int":   1,
						"float": 0.618,
						"s1":    "::ZipCode",
						"s2":    "::SN",
						"s3":    "::Latest",
						"s4":    "S4: $L(ZipCode)",
						"s5":    "S5: $L(SN)",
						"s6":    "S6: $L(Latest)",
						"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
					},
				},
			},

			map[string]interface{}{
				"int":   1,
				"float": 0.618,
				"s1":    "::ZipCode",
				"s2":    "::SN",
				"s3":    "::Latest",
				"s4":    "S4: $L(ZipCode)",
				"s5":    "S5: $L(SN)",
				"s6":    "S6: $L(Latest)",
				"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
				"map": map[string]interface{}{
					"int":   1,
					"float": 0.618,
					"s1":    "::ZipCode",
					"s2":    "::SN",
					"s3":    "::Latest",
					"s4":    "S4: $L(ZipCode)",
					"s5":    "S5: $L(SN)",
					"s6":    "S6: $L(Latest)",
					"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
				},
				"arr": []interface{}{
					1, 0.618,
					"::ZipCode", "::SN", "::Latest", "S4: $L(ZipCode)", "S5: $L(SN)", "S6: $L(Latest)",
					"S7: $L(ZipCode) $L(SN) $L(Latest)",
					map[string]interface{}{
						"int":   1,
						"float": 0.618,
						"s1":    "::ZipCode",
						"s2":    "::SN",
						"s3":    "::Latest",
						"s4":    "S4: $L(ZipCode)",
						"s5":    "S5: $L(SN)",
						"s6":    "S6: $L(Latest)",
						"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
					},
				},
			},
			testVal{
				Name: "::SN",
				Desc: "ST: $L(ZipCode) $L(SN) $L(Latest)",
			},
		},

		"map": map[string]interface{}{
			"int":   1,
			"float": 0.618,
			"s1":    "::ZipCode",
			"s2":    "::SN",
			"s3":    "::Latest",
			"s4":    "S4: $L(ZipCode)",
			"s5":    "S5: $L(SN)",
			"s6":    "S6: $L(Latest)",
			"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
			"map": map[string]interface{}{
				"int":   1,
				"float": 0.618,
				"s1":    "::ZipCode",
				"s2":    "::SN",
				"s3":    "::Latest",
				"s4":    "S4: $L(ZipCode)",
				"s5":    "S5: $L(SN)",
				"s6":    "S6: $L(Latest)",
				"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
			},
			"arr": []interface{}{
				1, 0.618,
				"::ZipCode", "::SN", "::Latest", "S4: $L(ZipCode)", "S5: $L(SN)", "S6: $L(Latest)",
				"S7: $L(ZipCode) $L(SN) $L(Latest)",
				map[string]interface{}{
					"int":   1,
					"float": 0.618,
					"s1":    "::ZipCode",
					"s2":    "::SN",
					"s3":    "::Latest",
					"s4":    "S4: $L(ZipCode)",
					"s5":    "S5: $L(SN)",
					"s6":    "S6: $L(Latest)",
					"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
				},
				[]interface{}{
					1, 0.618,
					"::ZipCode", "::SN", "::Latest", "S4: $L(ZipCode)", "S5: $L(SN)", "S6: $L(Latest)",
					"S7: $L(ZipCode) $L(SN) $L(Latest)",
					map[string]interface{}{
						"int":   1,
						"float": 0.618,
						"s1":    "::ZipCode",
						"s2":    "::SN",
						"s3":    "::Latest",
						"s4":    "S4: $L(ZipCode)",
						"s5":    "S5: $L(SN)",
						"s6":    "S6: $L(Latest)",
						"s7":    "S7: $L(ZipCode) $L(SN) $L(Latest)",
					},
				},
			},
			"struct": testVal{
				Name: "::SN",
				Desc: "ST: $L(ZipCode) $L(SN) $L(Latest)",
			},
		},

		"struct": testVal{
			Name: "::SN",
			Desc: "ST: $L(ZipCode) $L(SN) $L(Latest)",
		},

		"structptr": &testVal{
			Name: "::SN",
			Desc: "ST: $L(ZipCode) $L(SN) $L(Latest)",
		},
	}

}
