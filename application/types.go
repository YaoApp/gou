package application

// Application the application interface
type Application interface {
	Walk(path string, handler func(root, filename string, isdir bool) error, patterns ...string) error
	Read(name string) ([]byte, error)
	Write(name string, content []byte) error
	Remove(name string) error
	Exists(name string) (bool, error)
	Watch(handler func(event string, name string), interrupt chan uint8) error
}

// Pack the application pack interface
type Pack interface {
	Decode(data []byte) ([]byte, error)
	Encode(data []byte) ([]byte, error)
}
