package lang

// New create a new YAO DSL
func New(name string, kind int) (*YAO, error) {
	return &YAO{}, nil
}

// Create a new YAO DSL from source
func Create(name string, kind int, source []byte) (*YAO, error) {
	return &YAO{}, nil
}

// Open open YAO DSL file
func Open(file string) (*YAO, error) {
	return &YAO{}, nil
}

// Save export jsonc text and save to file
func (yao *YAO) Save() error { return nil }

// SaveAs export jsonc text and save to file
func (yao *YAO) SaveAs(file string) error { return nil }

// Bytes to bytes
func (yao *YAO) Bytes() ([]byte, error) { return []byte{}, nil }

// Download download the JSON file from workshop to vendor
func (yao *YAO) Download() {}
