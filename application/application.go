package application

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application/disk"
	"github.com/yaoapp/gou/application/yaz"
	"github.com/yaoapp/kun/log"
	"gopkg.in/yaml.v3"
)

// App the application interface
var App Application = nil

// Load the application
func Load(app Application) {

	App = app

	// load the pack.yao
	if has, _ := app.Exists("pack.yao"); !has {
		return
	}

	v, err := app.Read("pack.yao")
	if err != nil {
		log.Warn("Application Load: %s", err.Error())
		return
	}

	pack := Pack{}
	err = Parse("pack.yao", v, &pack)
	if err != nil {
		log.Warn("Application Load Parse Pack: %s", err.Error())
		return
	}

	if pack.Environments == nil {
		return
	}

	for name, value := range pack.Environments {
		if os.Getenv(name) == "" {
			if err := os.Setenv(name, value); err != nil {
				log.Warn("Application Load Set ENV: %s", err.Error())
			}
		}
	}
}

// OpenFromDisk open the application from disk
func OpenFromDisk(root string) (Application, error) {
	return disk.Open(root)
}

// OpenFromYazFile open the application from the .yaz file
func OpenFromYazFile(file string, cipher yaz.Cipher) (Application, error) {
	return yaz.OpenFile(file, cipher)
}

// OpenFromYazCache open the application from the cache
func OpenFromYazCache(file string, cipher yaz.Cipher) (Application, error) {
	return yaz.OpenCache(file, cipher)
}

// OpenFromYaz open the application from the binary .app file
func OpenFromYaz(reader io.Reader, file string, cipher yaz.Cipher) (Application, error) {
	return yaz.Open(reader, file, cipher)
}

// OpenFromDB open the application from database
func OpenFromDB(setting interface{}) (Application, error) {
	return nil, nil
}

// OpenFromStore open the application from the store driver
func OpenFromStore(setting interface{}) (Application, error) {
	return nil, nil
}

// OpenFromRemote open the application from the remote source server support .pkg | .app
func OpenFromRemote(url string, auth interface{}) (Application, error) {
	return nil, nil
}

// Parse the yao/json/jsonc/yaml type data
func Parse(name string, data []byte, vPtr interface{}) error {
	ext := filepath.Ext(name)
	switch ext {
	case ".yao", ".jsonc":
		content := trim(data, nil)
		err := jsoniter.Unmarshal(content, vPtr)
		if err != nil {
			return fmt.Errorf("[Parse] %s Error %s", name, err.Error())
		}
		return nil

	case ".json":
		err := jsoniter.Unmarshal(data, vPtr)
		if err != nil {
			return fmt.Errorf("[Parse] %s Error %s", name, err.Error())
		}
		return nil

	case ".yml", ".yaml":
		err := yaml.Unmarshal(data, vPtr)
		if err != nil {
			return fmt.Errorf("[Parse] %s Error %s", name, err.Error())
		}
		return nil
	}

	return fmt.Errorf("[Parse] %s Error %s does not support", name, ext)
}

// trim strips out comments and trailing commas and convert the input to a
// valid JSON per the official spec: https://tools.ietf.org/html/rfc8259
//
// The resulting JSON will always be the same length as the input and it will
// include all of the same line breaks at matching offsets. This is to ensure
// the result can be later processed by a external parser and that that
// parser will report messages or errors with the correct offsets.
func trim(src, dst []byte) []byte {
	dst = dst[:0]
	for i := 0; i < len(src); i++ {
		if src[i] == '/' {
			if i < len(src)-1 {
				if src[i+1] == '/' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src); i++ {
						if src[i] == '\n' {
							dst = append(dst, '\n')
							break
						} else if src[i] == '\t' || src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
				if src[i+1] == '*' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src)-1; i++ {
						if src[i] == '*' && src[i+1] == '/' {
							dst = append(dst, ' ', ' ')
							i++
							break
						} else if src[i] == '\n' || src[i] == '\t' ||
							src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
			}
		}
		dst = append(dst, src[i])
		if src[i] == '"' {
			for i = i + 1; i < len(src); i++ {
				dst = append(dst, src[i])
				if src[i] == '"' {
					j := i - 1
					for ; ; j-- {
						if src[j] != '\\' {
							break
						}
					}
					if (j-i)%2 != 0 {
						break
					}
				}
			}
		} else if src[i] == '}' || src[i] == ']' {
			for j := len(dst) - 2; j >= 0; j-- {
				if dst[j] <= ' ' {
					continue
				}
				if dst[j] == ',' {
					dst[j] = ' '
				}
				break
			}
		}
	}
	return dst
}
