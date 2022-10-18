package lang

// Dict the language dictionary
type Dict struct {
	Name    string
	Global  Words
	Widgets map[string]Words
}

// Words the language words
type Words map[string]string

// Lang the language interface
type Lang interface {
	Lang(trans func(widgetName string, inst string, value *string) bool)
}
