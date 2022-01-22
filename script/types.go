package script

import "io"

// VM 脚本接口
type VM interface {
	Compile(script *Script) error
	WithSID(sid string) VM
	WithGlobal(global map[string]interface{}) VM
	WithProcess(allow ...string) VM
	Run(name string, method string, args ...interface{}) (interface{}, error)
	RunScript(script *Script, method string, args ...interface{}) (interface{}, error)
	Load(filename string, name string) error
	Has(name string) bool
	MustLoad(filename string, name string) VM
	LoadSource(filename string, input io.Reader, name string) error
	MustLoadSource(filename string, input io.Reader, name string) VM
	Get(name string) (*Script, error)
	MustGet(name string) *Script
}

// Script 脚本
type Script struct {
	File      string
	Source    string
	Functions map[string]Function
}

// Function 脚本函数
type Function struct {
	Name      string
	NumOfArgs int
	Line      int
	Compiled  interface{}
}
