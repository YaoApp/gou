package gou

// MakeModel make a model instance
func MakeModel() *Model {
	return &Model{}
}

// DSLCompile compile the DSL
func (mod *Model) DSLCompile(source map[string]interface{}) error {
	// utils.Dump(source)
	return nil
}

// DSLCheck check the DSL
func (mod *Model) DSLCheck() error { return nil }

// DSLRefresh refresh the DSL
func (mod *Model) DSLRefresh() error { return nil }

// DSLRegister register the DSL
func (mod *Model) DSLRegister() error { return nil }

// DSLChange on the DSL file change
func (mod *Model) DSLChange(file string, event int) error { return nil }

// DSLDependencies ?get the dependencies of the DSL
func (mod *Model) DSLDependencies() ([]string, error) { return nil, nil }
