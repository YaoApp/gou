package gou

// Template the Template DSL
type Template struct{}

// MakeTemplate make a template instance
func MakeTemplate() *Template {
	return &Template{}
}

// DSLCompile compile the DSL
func (tpl *Template) DSLCompile(source map[string]interface{}) error { return nil }

// DSLCheck check the DSL
func (tpl *Template) DSLCheck() error { return nil }

// DSLRefresh refresh the DSL
func (tpl *Template) DSLRefresh() error { return nil }

// DSLRegister register the DSL
func (tpl *Template) DSLRegister() error { return nil }

// DSLChange on the DSL file change
func (tpl *Template) DSLChange(file string, event int) error { return nil }

// DSLDependencies get the dependencies of the DSL
func (tpl *Template) DSLDependencies() ([]string, error) { return nil, nil }
