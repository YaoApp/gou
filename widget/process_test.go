package widget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
)

func TestWidgetReload(t *testing.T) {
	load(t)
	v, err := process.New("widgets.dyform.Reload", "pad", "{}").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, nil, v)
}

func TestWidgetCustomProcess(t *testing.T) {
	load(t)
	v, err := process.New("widgets.dyform.Save", "pad", "foo").Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(v).Map()
	assert.Equal(t, "pad", res.Get("instance"))
	assert.Equal(t, "foo", res.Get("payload"))
}

// func moduleRegister() ModuleRegister {
// 	return ModuleRegister{
// 		"Models": func(name string, _ []byte) error {
// 			fmt.Printf("Model %s Registered\n", name)
// 			return nil
// 		},
// 		"Flows": func(name string, _ []byte) error {
// 			fmt.Printf("Flow %s Registered\n", name)
// 			return nil
// 		},
// 		"Apis": func(name string, _ []byte) error {
// 			fmt.Printf("API %s Registered\n", name)
// 			return nil
// 		},
// 	}
// }

// func load(t *testing.T) *Widget {
// 	root := os.Getenv("GOU_TEST_APP_ROOT")
// 	path := filepath.Join(root, "widgets", "dyform")
// 	w, err := LoadWidget(path, "dyform", moduleRegister())
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	err = w.Load()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	return w
// }
