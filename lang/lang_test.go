package lang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestSomeWidget struct {
	Flow  string
	Model string
	Hello string
	Foo   string
	Bar   string
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
	assert.Len(t, dict.Widgets, 2)
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
