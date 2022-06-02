package lang

// New create a new LString
func New(content string, langs map[string]string) LString {
	return LString{Content: content, Langs: langs}
}

// Get get a string
func (lstr LString) Get(lang string) string {
	if str, has := lstr.Langs[lang]; has {
		return str
	}
	return lstr.Content
}
