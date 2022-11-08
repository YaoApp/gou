package service

// *******************************************************
// * DEPRECATED											 *
// *******************************************************

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/takama/daemon"
	"github.com/yaoapp/kun/log"
)

// Load load the service config
func Load(name string, input []byte) (*Service, error) {
	var service Service
	err := jsoniter.Unmarshal(input, &service)
	if err != nil {
		return nil, err
	}

	kind := daemon.SystemDaemon
	if runtime.GOOS == "darwin" {
		kind = daemon.UserAgent
	}

	dependencies := []string{}
	if service.Requires != nil {
		dependencies = append(dependencies, service.Requires...)
	}
	if service.After != nil {
		dependencies = append(dependencies, service.After...)
	}
	service.name = name
	service.Daemon, err = daemon.New(service.name, service.Description, kind, dependencies...)
	if err != nil {
		return nil, err
	}

	err = service.SetTemplate()
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// Install service
func (service *Service) Install() (string, error) {
	return service.Daemon.Install(service.Args...)
}

// SetTemplate sets service config template
func (service *Service) SetTemplate() error {

	switch runtime.GOOS {
	case "darwin":
		return service.darwinTemplate()
	case "freebsd":
		return service.darwinTemplate()
	case "linux":
		return service.linuxTemplate()
	default:
		message := "service %s does not support windows."
		log.Error(message, service.name)
		return fmt.Errorf(message, service.name)
	}
}

// darwinTemplate MacOS service config
func (service *Service) darwinTemplate() error {
	tmpl := service.GetTemplate()
	cmd := service.Command
	if service.Process != "" {
		workdir := os.Getenv("YAO_ROOT")
		if workdir == "" {
			path, err := os.Getwd()
			if err != nil {
				workdir = path
			}
		}
		service.WorkDir = workdir
		cmd = fmt.Sprintf("yao run %s", service.Process)
	}

	if service.WorkDir != "" {
		tmpl = strings.ReplaceAll(tmpl, "<string>/usr/local/var</string>", fmt.Sprintf("<string>%s</string>", service.WorkDir))
	}

	if cmd != "" {
		tmpl = strings.ReplaceAll(tmpl, "<string>{{.Path}}</string>", fmt.Sprintf("<string>%s</string>", cmd))
	}

	if service.LogError != "" {
		tmpl = strings.ReplaceAll(tmpl, "<string>/usr/local/var/log/{{.Name}}.err</string>", fmt.Sprintf("<string>%s</string>", service.LogError))
	}

	if service.LogOutput != "" {
		tmpl = strings.ReplaceAll(tmpl, "<string>/usr/local/var/log/{{.Name}}.log</string>", fmt.Sprintf("<string>%s</string>", service.LogOutput))
	}

	return service.Daemon.SetTemplate(tmpl)
}

// darwinTemplate FreeBSD service config
func (service *Service) freebsdTemplate() error {
	message := "service %s does not support freebsd."
	log.Error(message, service.name)
	return fmt.Errorf(message, service.name)
}

// darwinTemplate Linux service config
func (service *Service) linuxTemplate() error {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return service.systemdTemplate()
	}
	if _, err := os.Stat("/sbin/initctl"); err == nil {
		return service.upstartTemplate()
	}
	return service.systemvTemplate()
}

// linuxTemplate Linux service systemd config
func (service *Service) systemdTemplate() error {
	tmpl := `[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart={{.Path}} {{.Args}}
Restart=on-failure
WorkingDirectory=/
User=root
Group=root
StandardOutput=file:/var/log/yao-{{.Name}}.log
StandardError=file:/var/log/yao-{{.Name}}-error.log

[Install]
WantedBy=multi-user.target`

	cmd := service.Command
	if service.Process != "" {
		workdir := os.Getenv("YAO_ROOT")
		if workdir == "" {
			path, err := os.Getwd()
			if err != nil {
				workdir = path
			}
		}
		service.WorkDir = workdir
		cmd = fmt.Sprintf("yao run %s", service.Process)
	}

	if cmd != "" {
		tmpl = strings.ReplaceAll(tmpl, "ExecStart={{.Path}} {{.Args}}", fmt.Sprintf("ExecStart=%s {{.Args}}", cmd))
	}

	if service.Restart != "" {
		tmpl = strings.ReplaceAll(tmpl, "Restart=on-failure", fmt.Sprintf("Restart=%s", service.Restart))
	}

	if service.WorkDir != "" {
		tmpl = strings.ReplaceAll(tmpl, "WorkingDirectory=/", fmt.Sprintf("WorkingDirectory=%s", service.WorkDir))
	}

	if service.LogOutput != "" {
		tmpl = strings.ReplaceAll(tmpl, "StandardOutput=file:/var/log/yao-{{.Name}}.log", fmt.Sprintf("StandardOutput=file:%s", service.LogOutput))
	}

	if service.LogError != "" {
		tmpl = strings.ReplaceAll(tmpl, "StandardError=file:/var/log/yao-{{.Name}}-error.log", fmt.Sprintf("StandardError=file:%s", service.LogError))
	}

	if service.User != "" {
		tmpl = strings.ReplaceAll(tmpl, "User=root", fmt.Sprintf("User=%s", service.User))
	}

	if service.Group != "" {
		tmpl = strings.ReplaceAll(tmpl, "Group=root", fmt.Sprintf("Group=%s", service.Group))
	}

	return service.Daemon.SetTemplate(tmpl)
}

// linuxTemplate Linux service upstart config
func (service *Service) upstartTemplate() error {
	message := "service %s does not support linux upstart, using systemd instead."
	log.Error(message, service.name)
	return fmt.Errorf(message, service.name)
}

// linuxTemplate Linux service systemv config
func (service *Service) systemvTemplate() error {
	message := "service %s does not support linux systemv, using systemd instead"
	log.Error(message, service.name)
	return fmt.Errorf(message, service.name)
}
