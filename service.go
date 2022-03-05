package gou

import (
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/service"
	"github.com/yaoapp/kun/exception"
)

// Services services loaded (Alpha)
var Services = map[string]*service.Service{}

// LoadService load socket server/client
func LoadService(source string, name string) (*service.Service, error) {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	config, err := helper.ReadFile(input)
	if err != nil {
		return nil, err
	}

	service, err := service.Load(name, config)
	if err != nil {
		return nil, err
	}
	Services[name] = service
	return service, nil
}

// SelectService Get socket by name
func SelectService(name string) *service.Service {
	service, has := Services[name]
	if !has {
		exception.New("Service:%s does not load", 500, name).Throw()
	}
	return service
}
