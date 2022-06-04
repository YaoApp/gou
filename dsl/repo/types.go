package repo

// API the git interface
type API interface {
	Content(file string) ([]byte, error)
	Dir(path string) ([]string, error)
}

// Repo the git repo
type Repo struct {
	Domain string
	Owner  string
	Repo   string
	Branch string
	Tag    string
	Commit string
	Call   API
}
