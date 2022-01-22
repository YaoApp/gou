module github.com/yaoapp/gou

go 1.13

require (
	github.com/buraksezer/olric v0.4.0 // indirect
	github.com/gin-gonic/gin v1.7.4
	github.com/go-errors/errors v1.4.1
	github.com/go-playground/validator/v10 v10.9.0 // indirect
	github.com/hashicorp/go-hclog v1.0.0
	github.com/hashicorp/go-plugin v1.4.3
	github.com/json-iterator/go v1.1.12
	github.com/robertkrimen/otto v0.0.0-20211024170158-b87d35c0b86f
	github.com/stretchr/testify v1.7.0
	github.com/ugorji/go v1.2.6 // indirect
	github.com/yaoapp/kun v0.6.5
	github.com/yaoapp/xun v0.6.0
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/sys v0.0.0-20211105183446-c75c47738b0c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	rogchap.com/v8go v0.7.0 // indirect
)

replace github.com/yaoapp/kun => ../kun

replace github.com/yaoapp/xun => ../xun

replace github.com/robertkrimen/otto => ../../otto
