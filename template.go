package gou

// Template the Template DSL
type Template struct{}

// MakeTemplate make a template instance
func MakeTemplate() *Template {
	return &Template{}
}

// DSLCompile compile the DSL
func (tpl *Template) DSLCompile(root string, file string, source map[string]interface{}) error {
	return nil
}

// DSLCheck check the DSL
func (tpl *Template) DSLCheck(source map[string]interface{}) error { return nil }

// DSLRefresh refresh the DSL
func (tpl *Template) DSLRefresh(root string, file string, source map[string]interface{}) error {
	return nil
}

// DSLRemove the DSL
func (tpl *Template) DSLRemove(root string, file string) error {
	return nil
}
