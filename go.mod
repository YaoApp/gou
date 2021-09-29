module github.com/yaoapp/gou

go 1.16

require (
	github.com/gin-gonic/gin v1.7.4
	github.com/go-playground/validator/v10 v10.9.0 // indirect
	github.com/hashicorp/go-hclog v0.16.2
	github.com/hashicorp/go-plugin v1.4.2
	github.com/json-iterator/go v1.1.11
	github.com/robertkrimen/otto v0.0.0-20210614181706-373ff5438452
	github.com/stretchr/testify v1.7.0
	github.com/ugorji/go v1.2.6 // indirect
	github.com/yaoapp/kun v0.6.5
	github.com/yaoapp/xun v0.5.2
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// replace github.com/yaoapp/kun => ../kun
// replace github.com/yaoapp/xun => ../xun
