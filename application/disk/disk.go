package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/yaoapp/gou/application/ignore"
	"github.com/yaoapp/kun/log"
)

// Disk disk
type Disk struct {
	root    string // the app root
	watched sync.Map
}

var defaultPatterns = []string{"*.yao", "*.json", "*.jsonc", "*.yaml", "*.yml", "*.so", "*.dll", "*.js", "*.py", "*.ts", "*.wasm"}
var ignoreWatchPatterns = []string{"public", "data", "db", "logs", "dist"}

// Open the application
func Open(root string) (*Disk, error) {

	// with home dir
	if strings.HasPrefix(root, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("[disk.Open] %s %s", root, err.Error())
		}
		root = homedir + strings.TrimPrefix(root, "~")
	}

	path, err := filepath.Abs(root)

	if err != nil {
		return nil, fmt.Errorf("[disk.Open] %s %s", root, err.Error())
	}

	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("[disk.Open] %s %s", root, err.Error())
	}

	return &Disk{root: path, watched: sync.Map{}}, nil
}

// Glob the files by pattern
func (disk *Disk) Glob(pattern string) ([]string, error) {
	file, err := disk.abs(pattern)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(file)
	if err != nil {
		return nil, err
	}

	for i, match := range matches {
		matches[i] = strings.TrimPrefix(match, disk.root)
	}

	return matches, nil
}

// Walk traverse folders and read file contents
func (disk *Disk) Walk(root string, handler func(root, file string, isdir bool) error, patterns ...string) error {
	rootAbs, err := disk.abs(root)
	if err != nil {
		return err
	}

	if patterns == nil {
		patterns = defaultPatterns
	}

	return filepath.Walk(rootAbs, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error("[disk.Walk] %s %s", filename, err.Error())
			return err
		}

		isdir := info.IsDir()
		if patterns != nil && !isdir && len(patterns) > 0 && patterns[0] != "-" {
			notmatched := true
			basname := filepath.Base(filename)
			for _, pattern := range patterns {
				if matched, _ := filepath.Match(pattern, basname); matched {
					notmatched = false
					break
				}
			}

			if notmatched {
				return nil
			}
		}

		name := strings.TrimPrefix(filename, rootAbs)
		if name == "" && isdir {
			name = string(os.PathSeparator)
		}

		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "/.") || strings.HasPrefix(name, "\\.") {
			return nil
		}

		if !isdir {
			name = filepath.Join(root, name)
		}

		err = handler(root, name, isdir)
		if filepath.SkipDir == err || filepath.SkipAll == err {
			return err
		}

		if err != nil {
			log.Error("[disk.Walk] %s %s", filename, err.Error())
			return err
		}

		return nil
	})
}

// Read the file content
func (disk *Disk) Read(name string) ([]byte, error) {
	file, err := disk.abs(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}

// Write the file content
func (disk *Disk) Write(name string, data []byte) error {
	file, err := disk.abs(name)
	if err != nil {
		return err
	}

	path := filepath.Dir(file)
	os.MkdirAll(path, os.ModePerm)

	return os.WriteFile(file, data, 0644)
}

// Remove the file
func (disk *Disk) Remove(name string) error {
	file, err := disk.abs(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(file)
}

// Exists check if the file is exists
func (disk *Disk) Exists(name string) (bool, error) {
	file, err := disk.abs(name)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Watch the file changes
func (disk *Disk) Watch(handler func(event string, name string), interrupt chan uint8) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	shutdown := make(chan bool, 1)
	gitignore := ignore.Compile(filepath.Join(disk.root, ".gitignore"), ignoreWatchPatterns...)

	// Add path
	err = disk.Walk("/", func(root, file string, isdir bool) error {

		rel := strings.TrimPrefix(file, string(os.PathSeparator))
		if gitignore.MatchesPath(rel) {
			return nil
		}

		if isdir {

			filename, err := disk.abs(file)
			if err != nil {
				return err
			}

			err = watcher.Add(filename)
			if err != nil {
				return err
			}
			log.Info("[Watch] Watching: %s", filename)
			disk.watched.Store(filename, true)
		}
		return nil
	})

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-shutdown:
				log.Info("[Watch] handler exit")
				return

			case event, ok := <-watcher.Events:
				if !ok {
					interrupt <- 1
					return
				}

				basname := filepath.Base(event.Name)
				isdir := true
				if strings.Contains(basname, ".") {
					isdir = false
				}

				ignore := true
				if !isdir {
					for _, pattern := range defaultPatterns {
						if matched, _ := filepath.Match(pattern, basname); matched {
							ignore = false
						}
					}

					if ignore {
						log.Info("[Watch] IGNORE %s", strings.TrimPrefix(event.Name, disk.root))
						break
					}
				}

				events := strings.Split(event.Op.String(), "|")
				for _, eventType := range events {

					// ADD / REMOVE Watching dir
					if isdir {
						switch eventType {
						case "CREATE":
							log.Info("[Watch] Watching: %s", strings.TrimPrefix(event.Name, disk.root))
							watcher.Add(event.Name)
							disk.watched.Store(event.Name, true)
							break

						case "REMOVE":
							log.Info("[Watch] Unwatching: %s", strings.TrimPrefix(event.Name, disk.root))
							watcher.Remove(event.Name)
							disk.watched.Delete(event.Name)
							break
						}
						continue
					}

					file := strings.TrimPrefix(event.Name, disk.root)
					log.Info("[Watch] %s %s", eventType, file)
					handler(eventType, file)
				}
				break

			case err, ok := <-watcher.Errors:
				if !ok {
					interrupt <- 2
					return
				}
				log.Error("[Watch] Error: %s", err.Error())
				break
			}
		}
	}()

	for {
		select {
		case code := <-interrupt:
			shutdown <- true
			log.Info("[Watch] Exit(%d)", code)
			fmt.Println(color.YellowString("[Watch] Exit(%d)", code))
			return nil
		}
	}
}

func (disk *Disk) abs(root string) (string, error) {
	root = filepath.Join(disk.root, root)
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return root, nil
}
